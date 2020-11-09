package main

import (
	"net/http"
	"encoding/json"
	"sync"
	"io/ioutil"
	"time"
	"fmt"
	"strings"
	"context"
	"go.mongodb.org/mongo-driver/mongo"
        "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)




type Article struct {
	//ID         string `json:"id"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	Content    string `json:"content"`
	Timestamp  string `json:"timestamp"`
}

type ResponseID struct {
    ID   interface{}
}

type ArticleHandlers struct {
	
	client     *mongo.Client
	sync.Mutex
	store map[string]Article
}

func (h *ArticleHandlers) connectMongo() {

	var (
		mongoURL = "mongodb+srv://SinghKislay:DKRHbyS8@cluster0.clnng.mongodb.net/Articles?retryWrites=true&w=majority"
	)
 
	// Initialize a new mongo client with options
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURL))
 
	// Connect the mongo client to the MongoDB server
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
 
	// Ping MongoDB
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		fmt.Println("could not ping to mongo db service: %v", err)
		return
	}
	h.client = client
	
	fmt.Println("connected to nosql database")
	
 }


func (h *ArticleHandlers) articles(w http.ResponseWriter, r *http.Request){
	
	switch r.Method {
	case "GET":
		h.getArticles(w, r)
		return
	case "POST":
		h.postArticle(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("method not allowed"))
		return
	}

}

func (h *ArticleHandlers) postArticle(w http.ResponseWriter, r *http.Request){
	
	bodyBytes, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	ct := r.Header.Get("content-type")
	if ct != "application/json"{
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("need content type application/json, but got '%s'", ct)))
		return
	}

	var article Article
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = json.Unmarshal(bodyBytes, &article)
	//article.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	article.Timestamp = fmt.Sprintf("%s", time.Now().String())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	h.Lock()
	database := h.client.Database("Articles")
	articlesCollection := database.Collection("article")
	articleResult, err := articlesCollection.InsertOne(ctx, bson.D{
		{Key: "title", Value: article.Title},
		{Key: "subtitle", Value: article.Subtitle},
		{Key: "content", Value: article.Content},
		{Key: "timestamp", Value: article.Timestamp},
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.Unlock()
		return
	}
	
	var response ResponseID 
	response.ID = articleResult.InsertedID
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	defer h.Unlock()
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *ArticleHandlers) getArticles(w http.ResponseWriter, r *http.Request){
	
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	database := h.client.Database("Articles")
	articlesCollection := database.Collection("article")
	h.Lock()
	cursor, err := articlesCollection.Find(ctx, bson.M{})
	if err != nil {
		fmt.Println(err)

		w.WriteHeader(http.StatusInternalServerError)
		h.Unlock()
		return
	}
	var retrieved_article []bson.M
	if err = cursor.All(ctx, &retrieved_article); err != nil {
		
		w.WriteHeader(http.StatusInternalServerError)
		h.Unlock()
		return
	}
	
	
	h.Unlock()

	jsonBytes, err := json.Marshal(retrieved_article)
	if err != nil {
		
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}


func (h *ArticleHandlers) getArticle(w http.ResponseWriter, r *http.Request){
	parts := strings.Split(r.URL.String(), "/")
	if len(parts) != 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	slug := parts[2]
	

	h.Lock()
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	database := h.client.Database("Articles")
	articlesCollection := database.Collection("article")
	objID, _ := primitive.ObjectIDFromHex(slug)
	cursor, err := articlesCollection.Find(ctx, bson.M{"_id":objID})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		h.Unlock()
		return
	}
	var retrieved_article []bson.M
	if err = cursor.All(ctx, &retrieved_article); err != nil {
		
		w.WriteHeader(http.StatusInternalServerError)
		h.Unlock()
		return
	}
	
	jsonBytes, err := json.Marshal(retrieved_article)
	if err != nil {
		
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		h.Unlock()
		return
	}
	
	h.Unlock()
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)

	return
}

func (h *ArticleHandlers) searchArticle(w http.ResponseWriter, r *http.Request){
	query := r.URL.Query()
	filter := query.Get("q")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	database := h.client.Database("Articles")
	articlesCollection := database.Collection("article")
	
	cursor, err := articlesCollection.Find(ctx, bson.M{
											"$or": []bson.M{
												bson.M{"title": filter},
												bson.M{"subtitle": filter},
												bson.M{"content": filter},
											},
										})

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		
		return
	}
	var retrieved_article []bson.M
	if err = cursor.All(ctx, &retrieved_article); err != nil {
		
		w.WriteHeader(http.StatusInternalServerError)
		
		return
	}
	
	jsonBytes, err := json.Marshal(retrieved_article)
	if err != nil {
		
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		
		return
	}
	
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)

}



func newArticleHandlers() *ArticleHandlers{
	return &ArticleHandlers{}
}


func main(){
	ArticleHandlers := newArticleHandlers()
	ArticleHandlers.connectMongo()
	
	http.HandleFunc("/articles/search", ArticleHandlers.searchArticle)
	http.HandleFunc("/articles/", ArticleHandlers.getArticle)
	http.HandleFunc("/articles", ArticleHandlers.articles)
	
	
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		panic(err)
	}
}
