package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/luyaotsung/hipchat-go/hipchat"
)

type HipChatEvent struct {
	Event string           `json:"event"`
	Item  HipChatEventItem `json:"item"`
}

type HipChatEventItem struct {
	Message HipChatEventMessage `json:"message"`
	Room    HipChatRoom         `json:"room"`
}

type HipChatEventMessage struct {
	Date          string
	File          HipChatFile `json:"file,omitempty"`
	From          interface{}
	Message       string
	Color         string
	Type          string `json:"type,omitempty"`
	Id            string
	MessageFormat string `json:"message_format"`
}

type HipChatUser struct {
	Id      float64 `json:"id"`
	Mention string  `json:"mention_name"`
	Name    string  `json:"name"`
}

type HipChatFile struct {
	Name     string `json:"name"`
	Size     int    `json:"size"`
	ThumbUrl string `json:"thumb_url"`
	Url      string `json:"url"`
}

type HipChatRoom struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

var (
	AccessToken = flag.String("token", "", "Access Token")
	RoomID      = flag.String("id", "", "ID of Chart Room")
	MsgColor    = flag.String("color", "green", "Color of Message")
)

func writeToFile(f *os.File, sourceRoom HipChatRoom, sourceMessage HipChatEventMessage) error {

	fmt.Println("--- Bgin writeToFile ---")
	strFrom, ok := sourceMessage.From.(string)

	if !ok {
		str, err := json.Marshal(sourceMessage.From)
		if err != nil {
			log.Printf("Error: %v\n", err)
			return err
		}
		var user HipChatUser
		err = json.Unmarshal([]byte(str), &user)
		if err != nil {
			log.Printf("Error: %v\n", err)
			return err
		}
		strFrom = user.Name
	}

	Msg_Split := strings.Split(sourceMessage.Message, " ")

	var sendMsg string
	var buffer bytes.Buffer

	for value := 1; value < len(Msg_Split); value++ {
		buffer.WriteString(fmt.Sprintf("%s ", Msg_Split[value]))
	}
	SearchKey := buffer.String()

	if Msg_Split[0] == "/Search" {
		sendMsg = fmt.Sprintf("You want to SEARCH Keyword is %s", SearchKey)
	} else if Msg_Split[0] == "/Asset" {
		sendMsg = fmt.Sprintf("You want to SEARCH ID is %s", SearchKey)
	} else {
		sendMsg = fmt.Sprintf("Usage : <br> /Search [Keyword] <br> /Asset [Device ID] <br> /Help : display help message")
	}

	fmt.Printf("\n[%s|%s] %s: %s\n", sourceMessage.Date, sourceRoom.Name, strFrom, sourceMessage.Message)
	send_Notify(*AccessToken, *RoomID, sendMsg, *MsgColor)

	fmt.Println("--- End ---")

	msg := fmt.Sprintf("[%s|%s] %s: %s\n", sourceMessage.Date, sourceRoom.Name, strFrom, sourceMessage.Message)
	_, err := f.WriteString(msg)
	return err
}

type myReader struct {
	*bytes.Buffer
}

func handler(w http.ResponseWriter, r *http.Request, outFile *os.File) {
	var notifyEvent HipChatEvent

	//fmt.Println(s)

	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(r.Body)
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	bodyString := string(bodyBytes)

	fmt.Println("\n\n--- Begin ---")
	fmt.Println(bodyString)

	json.NewDecoder(r.Body).Decode(&notifyEvent)

	err := writeToFile(outFile, notifyEvent.Item.Room, notifyEvent.Item.Message)
	if err != nil {
		log.Printf("Error: %v\n", err)
		w.WriteHeader(200)
	} else {
		w.WriteHeader(500)
	}
}

func send_Notify(token string, id string, message string, color string) {

	c := hipchat.NewClient(token)

	notifRq := &hipchat.NotificationRequest{Message: message, Color: color, From: "Asset Management"}
	resp2, err := c.Room.Notification(id, notifRq)

	if err != nil {
		fmt.Printf("Error during room notification %q\n", err)
		fmt.Printf("Server returns %+v\n", resp2)
	}
}

func main() {
	flag.Parse()

	if *AccessToken == "" || *RoomID == "" {
		fmt.Println("Please Input Access Token ,Room ID")
		os.Exit(-1)
	} else {
		filePath := "hip.log"
		out, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("can not open the log file: %s, err: %v", filePath, err)
		}
		defer out.Close()
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			handler(w, r, out)
		})
		http.ListenAndServe(":8088", nil)
	}
}
