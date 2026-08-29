package mutation

import (
	"strings"
	"testing"
)

func TestImageSelectionProposalChangesOnlyNamedWorkloadContainer(t *testing.T) {
	origin := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: checkout
  namespace: shop
spec:
  template:
    spec:
      containers:
      - name: app
        image: nginxxx
      - name: sidecar
        image: busybox:1
`
	proposal, err := imageSelectionProposal(origin, `{"container":"app","image":"nginx:latest"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(proposal, "image: nginx:latest") || !strings.Contains(proposal, "image: busybox:1") {
		t.Fatalf("unexpected proposal: %s", proposal)
	}
}

func TestImageSelectionProposalRejectsUnknownContainer(t *testing.T) {
	_, err := imageSelectionProposal("apiVersion: v1\nkind: Pod\nmetadata:\n  name: p\nspec:\n  containers:\n  - name: app\n    image: bad", `{"container":"other","image":"nginx:latest"}`)
	if err == nil {
		t.Fatal("expected unknown container to be rejected")
	}
}
