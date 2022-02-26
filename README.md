This application build a server that returns crawling result from pchome and momo e-shop, which is the most popular electronic commerce platform in Taiwan.
The data is crawled applying multithread(parellelism), which enables the server to return results in multiple pages fast.

![image](https://github.com/bobyanggg/crawl_project/blob/main/resource/image/crawl_project.png)

# Get started:
1. First, mySQL or MariaDB installation is needed.
Chrome browser is needed for crawling Momo.
golang and gRPC needs to be installed.<br/>
golang: https://go.dev/<br/>
gRPC: https://pjchender.dev/golang/grpc-getting-started/<br/>
2. Clone this repository.
3. Launch MySQL or MariaDB you installed, create a database if you don’t have one, you can put the config of your database in ‘github.com/bobyanggg/ /config/sql.json’.
4. Run go mod tidy to download the dependencies needed.
5. Go to ‘github.com/bobyanggg/crawl_project/server’.
6. Run “go run main.go” to start the server
7. Go to ‘github.com/bobyanggg/crawl_project/client’.
8. Run “go run main.go [your keyword]” to start searching.
9. You'll se response in your terminal and data will be stored in DB if the keyword is never searched.
![image](https://github.com/bobyanggg/crawl_project/blob/main/resource/image/response.png)
10. You can modify configs in github.com/bobyanggg/crawl_project/config
  * worker: Number of jobs per website
  * maxProduct: Maximum results
  * sleepTime: Time sleep between jobs that avoids being detected as DDOS attack

Enjoy!!!
