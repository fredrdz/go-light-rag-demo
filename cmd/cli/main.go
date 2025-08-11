// main.go
// OpenAI LLM + ChromeM (vector) + BoltDB (KV) + MemGraph (graph)
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	lrag "github.com/MegaGrindStone/go-light-rag"
	"github.com/MegaGrindStone/go-light-rag/handler"
	"github.com/MegaGrindStone/go-light-rag/llm"
	"github.com/MegaGrindStone/go-light-rag/storage"
	"github.com/philippgille/chromem-go"
)

type store struct {
	storage.Neo4J
	storage.Chromem
	storage.Bolt
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	// llm: OpenAI
	openai := llm.NewOpenAI(
		os.Getenv("OPENAI_API_KEY"),
		"gpt-5-nano", // https://platform.openai.com/docs/pricing
		llm.Parameters{},
		logger,
	)

	// graph: Neo4j
	graph, err := storage.NewNeo4J(
		"bolt://localhost:7687",
		"",
		"")
	if err != nil {
		panic(err)
	}

	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second*30)
		defer closeCancel()

		if err := graph.Close(closeCtx); err != nil {
			log.Printf("Error closing neo4jDB: %v\n", err)
		}
	}()

	// vector: ChromeM using OpenAI embeddings (or use chromem.NewEmbeddingFuncDefault()).
	emb := chromem.NewEmbeddingFuncOpenAI(
		os.Getenv("OPENAI_API_KEY"), chromem.EmbeddingModelOpenAI3Small) // chromem EmbeddingFunc

	vec,
		err := storage.NewChromem(
		"tmp/vec.db",
		5,
		storage.EmbeddingFunc(emb),
	) // file-backed, no server
	if err != nil {
		panic(err)
	}

	// kv: BoltDB
	kv, err := storage.NewBolt("tmp/kv.db") // file-backed, no server
	if err != nil {
		panic(err)
	}

	st := store{Neo4J: graph, Chromem: vec, Bolt: kv}
	h := handler.Default{
		ChunkMaxTokenSize: 1500,
		EntityTypes: []string{
			"character", "organization", "location", "time period", "object", "theme", "event",
		},
		Config: handler.DocumentConfig{
			MaxRetries:       5,
			BackoffDuration:  3 * time.Second,
			ConcurrencyCount: 5,
		},
	}

	// insert
	doc := lrag.Document{
		ID:      "doc-1",
		Content: "Neo4j stores entities; ChromeM stores vectors; Bolt stores chunks.",
	}

	err = lrag.Insert(doc, h, st, openai, logger)
	if err != nil {
		panic(err)
	}

	// query
	q := []lrag.QueryConversation{
		{
			Role:    lrag.RoleUser,
			Message: "where are graph and vectors stored?",
		},
	}

	result, err := lrag.Query(q, h, st, openai, logger)
	if err != nil {
		panic(err)
	}

	// access retrieved context
	fmt.Printf("Found %d local entities and %d global entities\n",
		len(result.LocalEntities), len(result.GlobalEntities))

	// print the result
	fmt.Println(result)
}
