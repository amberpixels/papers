module github.com/amberpixels/peppers

go 1.23

require (
	github.com/alecthomas/kong v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/jomei/notionapi v1.13.2
	github.com/yuin/goldmark v1.7.8
)

// Switching to custom fork for now
replace github.com/jomei/notionapi => github.com/amberpixels/notionapi v0.0.0-20241220211835-9cbb5232d733
