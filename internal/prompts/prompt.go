package prompts

const (
	Mutation_prompt   string = "Take the following in k8sgpt result %s as a guide to re-write this manifest fix a fixed version (you may make reasonable changes, e.g., fixing an image name or broken value etc..): %s  and respond with just the new manifest as a string without yaml or backticks around it. If you cannot make a suggestion for remediation, return {null} only, otherwise the response must be a working manifest (no partial responses)."
	Deployment_prompt string = "Take the following pod manifest %s make changes to the following deployment to produce this type of pod %s. Respond with just the a new valid manifest as a string without yaml or backticks around it. If you cannot make a suggestion for remediation, return {null} only, otherwise the response must be a working manifest (no partial responses)."
)
