module github.com/amberpixels/peppers

go 1.23

require (
	github.com/alecthomas/kong v1.4.0
	github.com/joho/godotenv v1.5.1
	github.com/jomei/notionapi v1.13.2
	github.com/yuin/goldmark v1.7.8
)

// Switching to custom fork for now
replace github.com/jomei/notionapi => github.com/amberpixels/notionapi v0.0.0-20241117224357-7e60e5d74d21
