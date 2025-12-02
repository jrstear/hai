package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"hai/storage"
)

var (
	esURL = flag.String("es-url", "", "Elasticsearch URL (default: from ELASTICSEARCH_URL env var)")
)

func main() {
	flag.Parse()

	url := *esURL
	if url == "" {
		url = os.Getenv("ELASTICSEARCH_URL")
		if url == "" {
			log.Fatal("Error: Elasticsearch URL not provided. Use -es-url flag or set ELASTICSEARCH_URL environment variable")
		}
	}

	ctx := context.Background()
	esStorage, err := storage.NewElasticsearchStorage(url)
	if err != nil {
		log.Fatalf("Failed to create Elasticsearch storage: %v", err)
	}
	defer esStorage.Close()

	// Get all embeddings
	embeddings, err := esStorage.ListAllEmbeddings(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to load embeddings: %v", err)
	}

	if len(embeddings) < 2 {
		log.Fatalf("Need at least 2 embeddings to analyze, found %d", len(embeddings))
	}

	// Get all speakers (centroids)
	speakers, err := esStorage.ListSpeakers(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to load speakers: %v", err)
	}

	fmt.Printf("Analyzing similarity scores:\n")
	fmt.Printf("  - Total embeddings: %d\n", len(embeddings))
	fmt.Printf("  - Total speakers (centroids): %d\n\n", len(speakers))

	// For each embedding, find best match against all centroids
	type similarityResult struct {
		EmbeddingID string
		BestMatch   float64
		SpeakerID   string
	}

	results := make([]similarityResult, 0, len(embeddings))
	allSimilarities := make([]float64, 0)

	for _, emb := range embeddings {
		if len(speakers) == 0 {
			continue
		}

		// Find best match against all centroids
		bestSimilarity := 0.0
		bestSpeakerID := ""
		for _, speaker := range speakers {
			similarity := storage.CosineSimilarity(emb.Embedding, speaker.Embedding)
			allSimilarities = append(allSimilarities, similarity)
			if similarity > bestSimilarity {
				bestSimilarity = similarity
				bestSpeakerID = speaker.ID
			}
		}

		results = append(results, similarityResult{
			EmbeddingID: emb.ID,
			BestMatch:   bestSimilarity,
			SpeakerID:   bestSpeakerID,
		})
	}

	// Sort results by similarity
	sort.Slice(results, func(i, j int) bool {
		return results[i].BestMatch > results[j].BestMatch
	})

	// Sort all similarities
	sort.Float64s(allSimilarities)

	// Print statistics
	fmt.Printf("Similarity Statistics:\n")
	fmt.Printf("  - Min: %.4f\n", allSimilarities[0])
	fmt.Printf("  - Max: %.4f\n", allSimilarities[len(allSimilarities)-1])
	fmt.Printf("  - Median: %.4f\n", allSimilarities[len(allSimilarities)/2])
	if len(allSimilarities) > 0 {
		mean := 0.0
		for _, s := range allSimilarities {
			mean += s
		}
		mean /= float64(len(allSimilarities))
		fmt.Printf("  - Mean: %.4f\n", mean)
	}

	// Percentiles
	p50 := allSimilarities[len(allSimilarities)*50/100]
	p75 := allSimilarities[len(allSimilarities)*75/100]
	p90 := allSimilarities[len(allSimilarities)*90/100]
	p95 := allSimilarities[len(allSimilarities)*95/100]
	p99 := allSimilarities[len(allSimilarities)*99/100]

	fmt.Printf("\nPercentiles:\n")
	fmt.Printf("  - 50th (median): %.4f\n", p50)
	fmt.Printf("  - 75th: %.4f\n", p75)
	fmt.Printf("  - 90th: %.4f\n", p90)
	fmt.Printf("  - 95th: %.4f\n", p95)
	fmt.Printf("  - 99th: %.4f\n", p99)

	// Count matches at different thresholds
	thresholds := []float64{0.75, 0.80, 0.82, 0.85, 0.87, 0.90}
	fmt.Printf("\nMatches at different thresholds:\n")
	for _, thresh := range thresholds {
		count := 0
		for _, r := range results {
			if r.BestMatch >= thresh {
				count++
			}
		}
		percent := float64(count) * 100.0 / float64(len(results))
		fmt.Printf("  - >= %.2f: %d/%d (%.1f%%)\n", thresh, count, len(results), percent)
	}

	// Show top matches
	fmt.Printf("\nTop 20 best matches:\n")
	for i := 0; i < 20 && i < len(results); i++ {
		r := results[i]
		status := "BELOW"
		if r.BestMatch >= storage.SimilarityThreshold {
			status = "ABOVE"
		}
		fmt.Printf("  %d. %s -> %s: %.4f (%s threshold %.2f)\n",
			i+1, r.EmbeddingID[:12], r.SpeakerID[:12], r.BestMatch, status, storage.SimilarityThreshold)
	}

	// Show matches just below current threshold
	fmt.Printf("\nMatches just below current threshold (%.2f):\n", storage.SimilarityThreshold)
	belowCount := 0
	for _, r := range results {
		if r.BestMatch >= 0.80 && r.BestMatch < storage.SimilarityThreshold {
			fmt.Printf("  %s -> %s: %.4f (%.4f below threshold)\n",
				r.EmbeddingID[:12], r.SpeakerID[:12], r.BestMatch, storage.SimilarityThreshold-r.BestMatch)
			belowCount++
			if belowCount >= 20 {
				break
			}
		}
	}

	// Recommendation
	fmt.Printf("\nRecommendation:\n")
	// Find threshold where we get reasonable matches (e.g., 50-80% of embeddings match)
	for _, thresh := range []float64{0.82, 0.80, 0.78, 0.75} {
		count := 0
		for _, r := range results {
			if r.BestMatch >= thresh {
				count++
			}
		}
		percent := float64(count) * 100.0 / float64(len(results))
		if percent >= 30 && percent <= 80 {
			fmt.Printf("  Consider threshold %.2f: would match %d/%d (%.1f%%) embeddings\n",
				thresh, count, len(results), percent)
			break
		}
	}
}

