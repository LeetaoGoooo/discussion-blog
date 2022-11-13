package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	"discussionblog/core"
)

type Response[T any] struct {
	Code    int    `json:"code,omitempty"`
	Data    *T     `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

var api = core.NewApi(os.Getenv("USER_NAME"), os.Getenv("SOURCE_REPO"), os.Getenv("ACCESS_TOKEN"))

func FetchPost(number int) {

	discussion, err := api.FetchPost(number)
	if err != nil {
		fmt.Printf("获取 discussion 异常,%v", err)

		return
	}

	postTemplate := template.Must(template.New("post.md").ParseFiles("post.md"))

	fileName := string(discussion.Title) + ".md"
	post, err := os.Create(fileName)

	if err != nil {
		fmt.Printf("生成文件异常,%v", err)
		return

	}

	defer post.Close()

	var labels string

	for _, label := range discussion.Lables.Nodes {
		labels += string(label.Name) + ","
	}

	labels = labels[0 : len(labels)-1]

	var data = map[string]interface{}{
		"Title":  discussion.Title,
		"Labels": labels,
		"Date":   discussion.CreatedAt.Time.Format("2006-01-02"),
		"Body":   discussion.Body,
	}

	err = postTemplate.Execute(post, data)

	if err != nil {
		fmt.Printf("模板解析异常,%v", err)
		return
	}

	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("读取文件异常,%v", err)
		return
	}
	api.CreateFile(os.Getenv("USER_NAME"), os.Getenv("TARGET_REPO"), "source/_posts/"+fileName, content)

}

func RemovePost(number int) {

}

func main() {
	var action string
	var number int
	flag.StringVar(&action, "action", "modify", "操作类型")
	flag.IntVar(&number, "number", -1, "对应的 discussion 的 number 值")
	flag.Parse()

	fmt.Printf("接受 action 为 %s, number 为 :%d\n", action, number)

	switch action {
	case "locked":
		FetchPost(number)
	case "unlocked":
		RemovePost(number)
	default:
		fmt.Printf("暂不支持该操作类型:%s", action)
	}
}
