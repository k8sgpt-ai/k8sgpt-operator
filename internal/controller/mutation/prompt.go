package mutation

const (
	mutation_prompt string = "Take the following in k8sgpt result %s and apply it to this manifest %s and respond with just the new manifest as a string without yaml or backticks around it. If you cannot make a suggestion for remediation, return the original manifest"
)
