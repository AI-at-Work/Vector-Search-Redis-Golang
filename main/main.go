package main

import (
	"Test_Project_GO/openai"
	"Test_Project_GO/redis"
	"fmt"
	"log"
)

const (
	userID    = "08ca0f25-7cd1-4c5d-9aed-6f62c1d03bf0"
	sessionID = "5c3eb21d-a356-44b8-832f-5bc99eed1fb3"
)

func main() {
	client, err := redis.InitRedisChatVector()
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}
	fmt.Println("Connection established successfully!")

	//if err := redis.CreateChatSchemaInCache(client, userID); err != nil {
	//	log.Fatalf("Failed to create chat schema: %v", err)
	//}
	//fmt.Println("Schema creation successful!")
	//
	//if err := redis.AddData(client, userID, sessionID); err != nil {
	//	log.Fatalf("Failed to add data: %v", err)
	//}

	queryText := "Wearable technology, like smartwatches, is making it easier for people"
	vec, err := openai.ApiEmbedding(queryText)
	if err != nil {
		log.Fatalf("Failed to create embedding: %v", err)
	}

	docs, total, err := redis.SearchInVectorCache(userID, sessionID, vec, client)
	if err != nil {
		log.Fatalf("Failed to search in vector cache: %v", err)
	}

	fmt.Println("Search Results:")
	fmt.Println("Total Results: ", total)
	for i, doc := range docs {
		fmt.Printf("%d. %s\n", i+1, doc.Properties["chat"])
		fmt.Printf("%s\n\n", doc.Properties["vector_dist"])
	}
}
