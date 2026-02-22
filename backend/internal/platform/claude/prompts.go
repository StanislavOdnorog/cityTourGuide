package claude

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

func buildSystemPrompt(language string) string {
	langInstruction := "Write in English."
	if language == "ru" {
		langInstruction = "Write in Russian (русский язык)."
	}

	return fmt.Sprintf(`You are a storyteller for an ambient audio guide app. Users walk through the city with headphones and hear stories about places they pass.

Your goal: create an engaging, emotionally resonant story about a specific place. NOT a Wikipedia article — a story that makes the listener stop and look around differently.

%s

STORY STRUCTURE (follow exactly):
1. Anchor (1 sentence): Where we are and what this is
2. Hook (1 sentence): An unexpected fact or angle
3. Facts (2-4 short facts): Only historically verified information
4. Meaning (1 sentence): Why this matters or what it reveals

RULES:
- Length: 50-200 words (15-45 seconds when read aloud)
- Tone: warm, conversational storyteller — like a knowledgeable friend walking beside you
- No addresses, no "Welcome to...", no "Did you know..."
- Never make up facts. If uncertain, say "it is said that..." or "legend has it..."
- Use present tense for descriptions of the place
- Use sensory details: what you'd see, hear, feel standing there

STORY LAYER TYPES (choose the most fitting one):
- atmosphere: How this place feels — sounds, smells, mood, energy
- human_story: Who lived, loved, struggled here — personal stories
- hidden_detail: Bullet marks, old signs, architectural details most people miss
- time_shift: "Stand here 100 years ago..." — temporal contrast

OUTPUT FORMAT: Respond with ONLY a JSON object (no markdown, no extra text):
{"text": "Your story text here", "layer_type": "one_of_the_types_above", "confidence": 80}

confidence: 0-100, how confident you are in the factual accuracy. Use 90+ only for well-documented facts. Use 60-80 for lesser-known places.`, langInstruction)
}

func buildUserPrompt(poi *domain.POI, language string) string {
	var sb strings.Builder

	sb.WriteString("Generate a story about this place:\n\n")
	sb.WriteString(fmt.Sprintf("Name: %s\n", poi.Name))

	if poi.NameRu != nil && *poi.NameRu != "" {
		sb.WriteString(fmt.Sprintf("Name (Russian): %s\n", *poi.NameRu))
	}

	sb.WriteString(fmt.Sprintf("Type: %s\n", poi.Type))
	sb.WriteString(fmt.Sprintf("Coordinates: %.6f, %.6f\n", poi.Lat, poi.Lng))

	if poi.Address != nil && *poi.Address != "" {
		sb.WriteString(fmt.Sprintf("Address: %s\n", *poi.Address))
	}

	if len(poi.Tags) > 2 { // not "{}" or "null"
		var tags map[string]interface{}
		if err := json.Unmarshal(poi.Tags, &tags); err == nil && len(tags) > 0 {
			relevantTags := extractRelevantTags(tags)
			if relevantTags != "" {
				sb.WriteString(fmt.Sprintf("Additional info: %s\n", relevantTags))
			}
		}
	}

	sb.WriteString("\nCity: Tbilisi, Georgia")

	if language == "ru" {
		sb.WriteString("\n\nWrite the story in Russian.")
	}

	return sb.String()
}

func extractRelevantTags(tags map[string]interface{}) string {
	relevant := []string{"wikidata", "wikipedia", "website", "description", "historic", "architect", "start_date", "building:levels"}
	var parts []string

	for _, key := range relevant {
		if val, ok := tags[key]; ok {
			parts = append(parts, fmt.Sprintf("%s: %v", key, val))
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "; ")
}
