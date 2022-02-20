This application build a server that returns crawling result from pchome and momo e-shop, which is the most popular electronic commerce platform in Taiwan.
The data is crawled applying multithread(concurrency), which enables the server to return results in multiple pages fast.

# Get started:
1. First, mySQL or MariaDB installation is needed.
Chrome browser is needed for crawling Momo.
golang and gRPC needs to be installed.<br/>
golang: https://go.dev/<br/>
gRPC: https://pjchender.dev/golang/grpc-getting-started/<br/>
2. Clone this repository.
3. Launch MySQL or MariaDB you installed, create a database if you don’t have one, you can put the config of your database in ‘github.com/bobyanggg/ /config/sql.json’
4. Run go mod tidy to download the dependencies needed.
5. Go to ‘github.com/bobyanggg/crawl_project/server’
6. Run “go run main.go” to start the server
7. Go to ‘github.com/bobyanggg/crawl_project/client’
8. Run “go run main.go [your keyword]” to start searching

Enjoy!!!
