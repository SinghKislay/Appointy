# Appointy
Assignment for placement

**Run Server**

``cd Appointy``

``go run main.go``

**Host name**

localhost:8081

eg requests:

**GET**

``curl localhost:8081/articles``

**GET by ID**

``curl localhost:8081/articles/5fa9654aa38356473a5dfabc``

**POST**

``curl localhost:8081/articles -X POST -d '{"title":"GO", "subtitle":"Go is an amazing language", "content":"Go has high emphasis on concurrency"}' -H "Content-Type: application/json"``