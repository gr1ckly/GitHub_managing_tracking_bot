package proxy

type Config struct {
	DB               string
	Port             string
	CoderAccessToken string
	TokenQueryParam  string
	PathPrefix       string
}
