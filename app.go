package main

import (
	"bufio"
	"bytes"
	"fmt"
	"fuliimg/goimghdr"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

// 定义当前目录
var APP_PATH, _ = os.Getwd()

// 定义 图片目录
var IMG_PATH = path.Join(filepath.Dir(APP_PATH), "fuliimages")

const url = "https://fuliba2021.net/flhz"

var reg = regexp.MustCompile(`(.*?)福利汇总第(.*?)期`)

type MyFormatter struct{}

func (s *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Local().Format("0000/00/00 00:00:00")
	msg := fmt.Sprintf("%s [%s] %s\n", timestamp, strings.ToUpper(entry.Level.String()), entry.Message)
	return []byte(msg), nil
}

func init() {
	APP_PATH = strings.Replace(APP_PATH, "\\", "/", -1)
	IMG_PATH = strings.Replace(IMG_PATH, "\\", "/", -1)

	//logger.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetFormatter(new(MyFormatter))
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	println("----------当前运行路径: " + APP_PATH + " ----------")
	println("----------图片存储路径: " + IMG_PATH + " ----------")
}

// 获取 列表页
func get_list(list_index int) {
	list_url := url
	if list_index > 1 {
		list_url = fmt.Sprintf("%s%s%d", url, "/page/", list_index)
	}

	res, err := _request(list_url)
	if err != nil {
		logger.Error("列表页请求失败:" + list_url)
		//logger.Error(err)
		return
	}
	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		logger.Error("列表页解析失败:" + list_url)
		//logger.Error(err)
		return
	}
	logger.Info("列表页请求成功:" + list_url)

	els := doc.Find("h2 a")
	els.Each(func(i int, el *goquery.Selection) {
		// 准备 参数
		content_title, _ := el.Attr("title")
		page_url, _ := el.Attr("href")
		// 测试查看参数
		logger.Debug("content_title:" + content_title)
		// 开始调用 get_page
		get_page(page_url, content_title)
	})

}

// 获取 内容页
func get_page(page_url, page_title string) {

	res, err := _request(page_url)
	if err != nil {
		logger.Error("--内容页请求失败:" + page_url)
		//logger.Error(err)
		return
	}
	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		logger.Error("--内容页解析失败:" + page_url)
		//logger.Error(err)
		return
	}
	logger.Info("--内容页请求开始:" + page_url)

	els := doc.Find(".article-paging a")
	els.Each(func(i int, el *goquery.Selection) {
		// 准备 参数
		content_url, _ := el.Attr("href")
		content_text := el.Text()
		// 开始调用 get_page
		get_content(content_url, page_title, content_text)

	})
}

// 获取 内容 分页
func get_content(content_url, content_title, content_index string) {

	res, err := _request(content_url)
	if err != nil {
		logger.Error("----详情页请求失败:" + content_index + ":" + content_title)
		//logger.Error(err)
		return
	}
	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		logger.Error("----详情页解析失败:" + content_index + ":" + content_title)
		//logger.Error(err)
		return
	}
	logger.Info("----详情页请求成功:" + content_index + ":" + content_title)

	tag := reg.FindStringSubmatch(content_title)
	els := doc.Find(".article-content img")
	els.Each(func(i int, el *goquery.Selection) {
		// 准备 参数
		img_src, _ := el.Attr("src")
		if img_src == "" {
			logger.Error("----src为空:" + content_index + ":" + content_title)
			return
		}
		img_path := path.Join(IMG_PATH, tag[1], tag[2], content_index, path.Base(img_src))
		// 开始调用 save_img
		save_img(img_src, img_path)
	})

}

// 保存图片
func save_img(img_src, img_path string) {
	logger.Debug("------开始下载图片:" + img_src)
	// 检测文件已近下载过
	if _, err := os.Stat(img_path); err == nil {
		logger.Debug("--------图片已下载过:" + img_path)
		return
	}

	if matches, _ := filepath.Glob(img_path + "*"); len(matches) > 0 {
		logger.Debug("--------图片已下载过:" + img_path)
		return
	}

	img_folder := path.Dir(img_path)

	// 判断目录是否存在
	if _, err := os.Stat(img_folder); err != nil {
		err := os.MkdirAll(img_folder, os.ModePerm)
		if err != nil {
			logger.Error("--------文件夹创建失败:" + img_folder)
			//logger.Error(err)
			return
		}
	}

	response, err := _request(img_src)
	if err != nil {
		logger.Error("--------图片请求失败:" + img_src)
		//logger.Error(err)
		return
	}
	robots, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Error("--------图片读取失败:" + img_src)
		logger.Error("              地址:" + img_path)
		return
	}
	defer response.Body.Close()

	// 如果文件名没有后缀
	if path.Ext(img_src) == "" {
		ext, err := goimghdr.WhatFromReader(bytes.NewReader(robots))
		if err != nil {
			logger.Error("--------图片解析失败:" + img_src)
			logger.Error("              地址:" + img_path)
			return
		}
		if ext == "jpeg" {
			ext = "jpg"
		}
		img_path = img_path + "." + ext
	}

	file, err := os.Create(img_path)
	if err != nil {
		logger.Error("--------图片创建失败:" + img_src)
		logger.Error("              地址:" + img_path)
		return
	}
	_, err = io.Copy(file, bytes.NewReader(robots))
	if err != nil {
		logger.Error("--------图片保存失败:" + img_src)
		logger.Error("              地址:" + img_path)
		return
	}
	logger.Info("--------图片保存成功:" + img_src)
	logger.Info("              地址:" + img_path)
}

// 请求二进制
func _request(url string) (*http.Response, error) {
	client := &http.Client{}
	//提交请求
	reqest, err := http.NewRequest("GET", url, nil)
	//增加header选项
	reqest.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.121 Safari/537.36")

	if err != nil {
		return nil, err
	}
	//处理返回结果
	response, err := client.Do(reqest)

	if err != nil {
		return response, err
	}
	if response.StatusCode != 200 {
		return response, http.ErrMissingFile
	}

	return response, err
}

func main() {
	if len(os.Args) > 1 {
		get_page(os.Args[1], os.Args[2])
		logger.Info("手动下载完成")
	} else {
		for i := 1; i < 8; i++ {
			get_list(i)
		}
		logger.Info("请求完成,一小时后重试")
		time.Sleep(60 * 60 * time.Second)
		main()
	}
}

func main1() {

	imgPath := "d:\\111\\"
	imgUrl := "https://tva1.sinaimg.cn/large/007asALTgy1gl8ljm8thpj30mj0zttio.jpg"

	fileName := path.Base(imgUrl)

	res, err := http.Get(imgUrl)
	if err != nil {
		fmt.Println("A error occurred!")
		return
	}
	defer res.Body.Close()

	// 获得get请求响应的reader对象
	reader := bufio.NewReaderSize(res.Body, 32*1024)

	file, err := os.Create(imgPath + fileName)
	if err != nil {
		panic(err)
	}
	// 获得文件的writer对象
	writer := bufio.NewWriter(file)

	written, _ := io.Copy(writer, reader)
	fmt.Printf("Total length: %d", written)

}

func main2() {
	save_img("https://tva1.sinaimg.cn/large/007qbgWbgy1geaz4l0mqpj30m80tm4ea.jpg", "d:/111/007qbgWbgy1geaz4l0mqpj30m80tm4ea.jpg")
}
