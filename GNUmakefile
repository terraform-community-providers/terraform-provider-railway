default: testacc

# Run acceptance tests
.PHONY: testacc

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

download-schema:
	npx get-graphql-schema https://backboard.railway.app/graphql/v2 > schema.graphql
