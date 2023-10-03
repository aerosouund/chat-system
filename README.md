# Chat System

A chat system that has multiple Application identified by token, each Application has many chats identified by a number ( number should start from 1) , each Chat has many messages identified by a number ( number should start from 1)

- the endpoints should be RESTful
- Use MySQL as datastore
- use ElasticSearch for searching through messages of a specific chat
- use a Worker to create chats and messages when a message is sent on the queue

## Endpoints

```bash
curl -X POST http://localhost:8080/applications/ -d '{"name": "application1"}' -H "Content-Type: application/json"
```

- list applications

```bash
curl -X GET http://localhost:8080/applications/
```

- create chat for application1

```bash
curl -X POST http://localhost:8080/applications/dh82jm0q/chats
```

- list chats for application1
  
```bash
curl -X GET http://localhost:8080/applications/dh82jm0q/chats
```

- create message for chat 1

```bash
curl -X POST http://localhost:8080/api/v1/applications/dh82jm0q/chats/1/messages/ -d '{"body": "hello"}' -H "Content-Type: application/json"
```

- list all messages for chat 1

```bash
curl -X GET http://localhost:8080/applications/dh82jm0q/chats/1/messages/
```

- partial search in specific chat messages
  
```bash
curl -X GET "http://localhost:8080/applications/dh82jm0q/chats/6/messages/search?query=hel"
```
