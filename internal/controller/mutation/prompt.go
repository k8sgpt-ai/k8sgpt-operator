package mutation

const (
	mutation_prompt string = "Take the following in k8sgpt result %s as a guide to re-write this manifest so it doesn't break: %s  and respond with just the new manifest as a string without yaml or backticks around it. If you cannot make a suggestion for remediation, return {null} only"
)
