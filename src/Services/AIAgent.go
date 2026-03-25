package Services

import (
	"context"
	"cuento-backend/src/Services/AIClients/GeminiClient"
)

type AIClient interface {
	Call(ctx context.Context, prompt string) (string, error)
}

type AIAgent struct {
	agentType string
	modelName string
	client    AIClient
}

func NewAIAgent(agentType string, apiKey string, modelName string) *AIAgent {
	switch agentType {
	case "gemini":
		return &AIAgent{
			agentType: agentType,
			modelName: modelName,
			client:    GeminiClient.NewGeminiClient(apiKey, modelName),
		}
	default:
		return nil
	}
}

//func (a *AIAgent) AnalyzeGuestMessage(msg string, guestName string) string {
//	systemInstruction := `You are the Cuento Forum Butler, an automated security and routing agent.
//Analyze the provided guest post and categorize it strictly into one of these types:
//- 'spam': Commercial ads, gibberish, or bot-generated links.
//- 'general_question': Inquiries about forum rules, technical help, or joining.
//- 'user_question': Questions directed at a specific player or character.
//- 'obscenity': Personal attacks, slurs, or real-world threats (Distinguish from fictional RPG roleplay).
//
//RULES:
//1. If 'spam', response must be null.
//2. If 'obscenity', the response must be a firm warning about site conduct.
//3. If 'user_question', attempt to extract the username for 'notify_user_id'.
//4. Return ONLY valid JSON. No markdown formatting.
//
//JSON Schema:
//{
//  "category": string,
//  "response": string|null,
//  "notify_user_id": string|null,
//  "notify_admins": boolean,
//  "confidence_score": float (0.0-1.0)
//}`
//
//}
