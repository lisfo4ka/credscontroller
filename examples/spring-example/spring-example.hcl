# Allow a token to get a secret from the generic secret backend
# for the client role.
path "secret/spring-example" {
  capabilities = ["read", "list"]
}
