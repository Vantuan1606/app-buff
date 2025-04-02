package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TikTokAccount struct {
	Username string `bson:"user"`
	Password string `bson:"pass"`
	// Cookies  string `bson:"cookies"`
}

const mongoURI = "mongodb+srv://kietkhan0:BfOAkPYYpzQJKLbM@marketingtool01.pe1d5ra.mongodb.net/?"

var videoURLPattern = regexp.MustCompile(`https?:\/\/www\.tiktok\.com\/@[\w.]+\/live`)
var maxActiveAccounts int
var watchDuration time.Duration

func getUserInput(prompt string) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(prompt)

		input, _ := reader.ReadString('\n')
		value, err := strconv.Atoi(input[:len(input)])

		if err == nil && value > 0 {
			return value
		}

		return value
		fmt.Println("Vui lòng nhập một số hợp lệ.")

	}
}

func getAccounts() ([]TikTokAccount, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.TODO())

	collection := client.Database("marketingTool01").Collection("marketingTool01	")
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var accounts []TikTokAccount
	for cursor.Next(context.TODO()) {
		var account TikTokAccount
		if err := cursor.Decode(&account); err != nil {
			log.Println("Lỗi khi decode account:", err)
			continue
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func watchVideoWithAccount(account TikTokAccount, videoURL string, wg *sync.WaitGroup, statusChan chan int) {
	defer wg.Done()

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.tiktok.com/login"),
		chromedp.SendKeys(`input[name="username"]`, account.Username),
		chromedp.SendKeys(`input[name="password"]`, account.Password),
		chromedp.Click(`button[type="submit"]`),
		chromedp.Sleep(5*time.Second),
		chromedp.Navigate(videoURL),
		chromedp.Sleep(watchDuration),
	)
	if err != nil {
		log.Println("Lỗi khi xem video với tài khoản", account.Username, err)
		statusChan <- http.StatusInternalServerError
	} else {
		log.Println("Tài khoản", account.Username, "đã xem xong video.")
		statusChan <- http.StatusOK
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Nhập URL video TikTok: ")
	videoURL, _ := reader.ReadString('\n')
	videoURL = videoURL[:len(videoURL)-1]

	if !videoURLPattern.MatchString(videoURL) {
		log.Fatal("URL video không hợp lệ.")
	}

	maxActiveAccounts = getUserInput("Nhập số lượng tài khoản để buff view: ")
	watchDuration = time.Duration(getUserInput("Nhập thời gian xem video (giây): ")) * time.Minute

	accounts, err := getAccounts()
	if err != nil {
		log.Fatal("Không thể lấy tài khoản:", err)
	}

	if len(accounts) > maxActiveAccounts {
		accounts = accounts[:maxActiveAccounts]
	}

	var wg sync.WaitGroup
	statusChan := make(chan int, len(accounts))

	for _, acc := range accounts {
		wg.Add(1)
		go watchVideoWithAccount(acc, videoURL, &wg, statusChan)
		fmt.Println("tài khoản  ", acc)
		time.Sleep(30 * time.Second)
	}

	wg.Wait()
	close(statusChan)

	status := http.StatusOK
	for s := range statusChan {
		if s != http.StatusOK {

			status = s
		}
	}

	fmt.Printf("Buff view hoàn thành cho video %s với status code: %d\n", status)
}
