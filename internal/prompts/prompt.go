package prompts

const (
	Mutation_prompt string = "Choose a safe image-only repair for this Kubernetes finding: %s\n\nManifest:\n%s\n\nReturn exactly one JSON object and nothing else: {\"container\":\"the existing container or init-container name\",\"image\":\"the corrected image reference\"}. Do not use Markdown, prose, a YAML manifest, or additional keys. The container must already exist and only its image may change."
)
