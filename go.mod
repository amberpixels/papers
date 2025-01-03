module github.com/amberpixels/peppers

go 1.23

require (
	github.com/alecthomas/kong v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/jomei/notionapi v1.13.2
	github.com/stretchr/testify v1.10.0
	github.com/yuin/goldmark v1.7.8
	golang.org/x/net v0.33.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Switching to custom fork for now
replace github.com/jomei/notionapi => github.com/amberpixels/notionapi v0.0.0-20241221003507-e57529ef2311
