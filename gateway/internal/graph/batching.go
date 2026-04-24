package graph

import "github.com/go-mirofish/go-mirofish/gateway/internal/memory"

func BuildEpisodes(text string, chunkSize int, chunkOverlap int, graphID string) []memory.Episode {
	chunks := memory.ChunkText(text, chunkSize, chunkOverlap)
	return memory.EncodeEpisodes(chunks, graphID)
}
