package redis

import (
	"Test_Project_GO/openai"
	"fmt"
	"github.com/RediSearch/redisearch-go/v2/redisearch"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/redis/rueidis"
	"strconv"
	"time"
)

const INDEX_NAME = "user-chat"

func InitRedisChatVector() (*redisearch.Client, error) {
	dbName, err := strconv.Atoi("0")
	if err != nil {
		return nil, fmt.Errorf("failed to parse database number: %w", err)
	}

	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379",
				redis.DialPassword("secrete_password"),
				redis.DialDatabase(dbName),
			)
		},
	}

	client := redisearch.NewClientFromPool(pool, INDEX_NAME)
	return client, nil
}

func CreateChatSchemaInCache(database *redisearch.Client, userID string) error {
	dims, err := strconv.Atoi("1536")
	if err != nil {
		return fmt.Errorf("failed to parse dimensions: %w", err)
	}

	database.SetIndexName(fmt.Sprintf("%s:%s", INDEX_NAME, userID))

	sc := redisearch.NewSchema(redisearch.DefaultOptions).
		AddField(redisearch.NewTextFieldOptions("session_id", redisearch.TextFieldOptions{NoStem: true, NoIndex: false})).
		AddField(redisearch.NewNumericFieldOptions("timestamp", redisearch.NumericFieldOptions{Sortable: true, NoIndex: false})).
		AddField(redisearch.NewTextFieldOptions("chat", redisearch.TextFieldOptions{NoStem: true, NoIndex: true})).
		AddField(redisearch.NewVectorFieldOptions("chat_embeddings", redisearch.VectorFieldOptions{
			Algorithm: redisearch.Flat,
			Attributes: map[string]interface{}{
				"DIM":             dims,
				"DISTANCE_METRIC": "COSINE",
				"TYPE":            "FLOAT32",
			},
		}))

	_ = database.Drop()

	if err := database.CreateIndex(sc); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

func AddToVectorCache(userID, sessionID string, timestamp int64, chat string, database *redisearch.Client) error {
	id := uuid.New()
	key := fmt.Sprintf("%s:%s:%s", INDEX_NAME, userID, id.String())

	vector, err := openai.ApiEmbedding(chat)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	doc := redisearch.NewDocument(key, 1.0)
	doc.Set("timestamp", timestamp).
		Set("session_id", sessionID).
		Set("chat", chat).
		Set("chat_embeddings", rueidis.VectorString32(vector))

	if err := database.Index([]redisearch.Document{doc}...); err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}

	return nil
}

func SearchInVectorCache(userID, sessionID string, userQueryEmbedding []float32, database *redisearch.Client) (docs []redisearch.Document, total int, err error) {
	//  FT.SEARCH user-chat:123123 "(@session_id:assdasd)=>[KNN 5 @chat_embeddings $vector AS vector_score]" PARAMS 2 vector "[0.2,0.3,0.4,...]" RETURN 3 session_id chat vector_score SORTBY vector_score DIALECT 2
	// @session_id:(%s) @chat_embeddings:[VECTOR_RANGE 0.2 $query_vector]=>{$YIELD_DISTANCE_AS: vector_dist}
	maxLimit := 5
	database.SetIndexName(fmt.Sprintf("%s:%s", INDEX_NAME, userID))

	userQueryParsed := rueidis.VectorString32(userQueryEmbedding)

	r := redisearch.Query{
		Raw: fmt.Sprintf("@session_id:(%s)=>[KNN 10 @chat_embeddings $query_vector AS vector_dist]", sessionID),
		Params: map[string]interface{}{
			"query_vector": userQueryParsed,
		},
		Dialect: 2,
		SortBy: &redisearch.SortingKey{
			Field: "vector_dist",
		},
		ReturnFields: []string{"chat", "vector_dist"},
	}
	query := r.Limit(0, maxLimit)
	docs, total, err = database.Search(query)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to perform search: %w", err)
	}

	fmt.Printf("Total: %d\n", total)
	fmt.Printf("Docs: %v\n", docs)
	return
}

func AddData(client *redisearch.Client, userID, sessionID string) error {
	data := []struct {
		category string
		texts    []string
	}{
		{
			category: "Technology and Innovation",
			texts: []string{
				"Artificial intelligence is transforming industries by automating tasks and enhancing decision-making.",
				"The rise of virtual reality has opened new possibilities for remote learning and immersive gaming.",
				"Self-driving cars are poised to revolutionize the way we commute, reducing accidents and traffic congestion.",
				"Wearable technology, like smartwatches, is making it easier for people to monitor their health metrics in real time.",
				"Advances in renewable energy technologies are crucial for combating climate change and reducing global reliance on fossil fuels.",
			},
		},
		{
			category: "Travel and Exploration",
			texts: []string{
				"Backpacking across Europe offers an intimate glimpse into the rich history and diverse cultures of the continent.",
				"The popularity of eco-tourism is helping to preserve natural habitats while providing travelers with unique experiences.",
				"Exploring the coral reefs of Australia is a breathtaking adventure that highlights the beauty of marine biodiversity.",
				"Historical landmarks, from the Great Wall of China to the pyramids of Egypt, tell stories of ancient civilizations.",
				"Culinary tours in cities like Paris and Tokyo allow travelers to explore the local flavors and culinary traditions.",
			},
		},
		{
			category: "Health and Wellness",
			texts: []string{
				"Regular exercise is key to maintaining physical health and improving overall well-being.",
				"Mental health is gaining recognition as a critical aspect of overall health, with mindfulness and meditation becoming more popular.",
				"The benefits of a balanced diet are well-documented, including improved energy levels and better immune system function.",
				"Sleep hygiene plays a crucial role in physical and mental health, impacting mood, cognition, and performance.",
				"The rise of telemedicine is making healthcare more accessible, allowing patients to consult with doctors remotely.",
			},
		},
		{
			category: "Education and Learning",
			texts: []string{
				"Online education platforms are expanding access to learning opportunities for people around the world.",
				"The integration of technology in classrooms is enhancing interactive learning and engagement among students.",
				"Lifelong learning is essential for career development and staying current with evolving industry trends.",
				"Critical thinking and problem-solving are fundamental skills that education systems aim to instill in students.",
				"Bilingual education has numerous cognitive benefits, including better memory and enhanced problem-solving skills.",
			},
		},
	}

	for _, category := range data {
		fmt.Printf("Adding data for category: %s\n", category.category)
		for _, text := range category.texts {
			if err := AddToVectorCache(userID, sessionID, time.Now().UnixMilli(), text, client); err != nil {
				return fmt.Errorf("failed to add data to vector cache for category %s: %w", category.category, err)
			}
			time.Sleep(time.Second) // Reduced sleep time for faster execution
		}
	}

	fmt.Println("All data loaded successfully!")
	return nil
}

/*
func SearchInVectorCache(userID, sessionID string, userQueryEmbedding []float32, database *redisearch.Client) (docs []redisearch.Document, total int, err error) {
	//  FT.SEARCH user-chat:123123 "(@session_id:assdasd)=>[KNN 5 @chat_embeddings $vector AS vector_score]" PARAMS 2 vector "[0.2,0.3,0.4,...]" RETURN 3 session_id chat vector_score SORTBY vector_score DIALECT 2

	maxLimit := 5
	database.SetIndexName(fmt.Sprintf("%s:%s", INDEX_NAME, userID))

	userQueryParsed := rueidis.VectorString32(userQueryEmbedding)

	res, err := redis.Values(database.GetConnection().Do(
		"FT.SEARCH", fmt.Sprintf("%s:%s", INDEX_NAME, userID),
		fmt.Sprintf("@session_id:%s @chat_embeddings:[VECTOR_RANGE 0.2 $query_vector]=>{$YIELD_DISTANCE_AS: vector_dist}", sessionID),
		"PARAMS", 2,
		"query_vector", userQueryParsed,
		"SORTBY", "vector_dist",
		"LIMIT", "0", maxLimit,
		"DIALECT", 2))
	if err != nil {
		return nil, -1, err
	}

	if total, err = redis.Int(res[0], nil); err != nil {
		return nil, -1, err
	}

	docs = make([]redisearch.Document, 0, len(res)-1)
	fmt.Println("LEN res: ", len(res))

	skip := 2
	scoreIdx := -1
	fieldsIdx := 1
	payloadIdx := 1
	if len(res) > skip {
		for i := 1; i < len(res); i += skip {
			if d, e := redisearch.LoadDocument(res, i, scoreIdx, payloadIdx, fieldsIdx); e == nil {
				docs = append(docs, d)
			} else {
				log.Print("Error parsing doc: ", e)
			}
		}
	}

	return

	//limit := 10
	//query := redisearch.Query{
	//	Raw: fmt.Sprintf("@session_id:%s=>[KNN 5 @chat_embeddings $vector AS score]", sessionID),
	//	Params: map[string]interface{}{
	//		"vector": userQueryParsed,
	//	},
	//	//SortBy: &redisearch.SortingKey{
	//	//	Field:     "timestamp",
	//	//	Ascending: false,
	//	//},
	//}
	//
	//docs, total, err = database.Search(query.Limit(0, limit))
	//if err != nil {
	//	return nil, -1, fmt.Errorf("failed to perform search: %w", err)
	//}
	//
	//fmt.Printf("Total: %d\n", total)
	//fmt.Printf("Docs: %v\n", docs)
	//return
}

*/
