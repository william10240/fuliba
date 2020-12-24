package main

import (
	"bufio"
	"bytes"
	"fmt"
	"fuliimg_go/goimghdr"
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

// _appPath 定义当前目录
var _appPath, _ = os.Getwd()

// _imgPath 定义 图片目录
var _imgPath = path.Join(filepath.Dir(_appPath), "fuliimages2")

const url = "https://fuliba2020.net/category/flhz"

var reg = regexp.MustCompile(`(.*?)福利汇总第(.*?)期`)

type myFormatter struct{}

func (s *myFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Local().Format("0000/00/00 00:00:00")
	msg := fmt.Sprintf("%s [%s] %s\n", timestamp, strings.ToUpper(entry.Level.String()), entry.Message)
	return []byte(msg), nil
}

func init() {
	_appPath = strings.Replace(_appPath, "\\", "/", -1)
	_imgPath = strings.Replace(_imgPath, "\\", "/", -1)

	//logger.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetFormatter(new(myFormatter))
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	println("----------当前运行路径: " + _appPath + " ----------")
	println("----------图片存储路径: " + _imgPath + " ----------")
}

// 获取 列表页
func getList(listIndex int) {
	listURL := url
	if listIndex > 1 {
		listURL = fmt.Sprintf("%s%s%d", url, "/page/", listIndex)
	}

	res, err := _request(listURL)
	if err != nil {
		logger.Error("列表页请求失败:" + listURL)
		//logger.Error(err)
		return
	}
	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		logger.Error("列表页解析失败:" + listURL)
		//logger.Error(err)
		return
	}
	logger.Info("列表页请求成功:" + listURL)

	els := doc.Find("h2 a")
	els.Each(func(i int, el *goquery.Selection) {
		// 准备 参数
		contentTitle, _ := el.Attr("title")
		pageURL, _ := el.Attr("href")
		// 测试查看参数
		logger.Debug("content_title:" + contentTitle)
		// 开始调用 get_page
		getPage(pageURL, contentTitle)
	})

}

// 获取 内容页
func getPage(pageURL, pageTitle string) {

	res, err := _request(pageURL)
	if err != nil {
		logger.Error("--内容页请求失败:" + pageURL)
		//logger.Error(err)
		return
	}
	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		logger.Error("--内容页解析失败:" + pageURL)
		//logger.Error(err)
		return
	}
	logger.Info("--内容页请求开始:" + pageURL)

	els := doc.Find(".article-paging a")
	els.Each(func(i int, el *goquery.Selection) {
		// 准备 参数
		contentURL, _ := el.Attr("href")
		contentText := el.Text()
		// 开始调用 get_page
		getContent(contentURL, pageTitle, contentText)

	})
}

// 获取 内容 分页
func getContent(contentURL, contentTitle, contentIndex string) {

	res, err := _request(contentURL)
	if err != nil {
		logger.Error("----详情页请求失败:" + contentIndex + ":" + contentTitle)
		//logger.Error(err)
		return
	}
	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		logger.Error("----详情页解析失败:" + contentIndex + ":" + contentTitle)
		//logger.Error(err)
		return
	}
	logger.Info("----详情页请求成功:" + contentIndex + ":" + contentTitle)

	tag := reg.FindStringSubmatch(contentTitle)
	els := doc.Find(".article-content img")
	els.Each(func(i int, el *goquery.Selection) {
		// 准备 参数
		imgSrc, _ := el.Attr("src")
		if imgSrc == "" {
			logger.Error("----src为空:" + contentIndex + ":" + contentTitle)
			return
		}
		imgPath := path.Join(_imgPath, tag[1], tag[2], contentIndex, path.Base(imgSrc))
		// 开始调用 save_img
		saveImg(imgSrc, imgPath)
	})

}

// 保存图片
func saveImg(imgSrc, imgPath string) {
	logger.Debug("------开始下载图片:" + imgSrc)
	// 检测文件已近下载过
	if _, err := os.Stat(imgPath); err == nil {
		logger.Debug("--------图片已下载过:" + imgPath)
		return
	}

	if matches, _ := filepath.Glob(imgPath + "*"); len(matches) > 0 {
		logger.Debug("--------图片已下载过:" + imgPath)
		return
	}

	imgFolder := path.Dir(imgPath)

	// 判断目录是否存在
	if _, err := os.Stat(imgFolder); err != nil {
		err := os.MkdirAll(imgFolder, os.ModePerm)
		if err != nil {
			logger.Error("--------文件夹创建失败:" + imgFolder)
			//logger.Error(err)
			return
		}
	}

	response, err := _request(imgSrc)
	if err != nil {
		logger.Error("--------图片请求失败:" + imgSrc)
		//logger.Error(err)
		return
	}
	robots, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Error("--------图片读取失败:" + imgSrc)
		logger.Error("              地址:" + imgPath)
		return
	}
	defer response.Body.Close()

	// 如果文件名没有后缀
	if path.Ext(imgSrc) == "" {
		ext, err := goimghdr.WhatFromReader(bytes.NewReader(robots))
		if err != nil {
			logger.Error("--------图片解析失败:" + imgSrc)
			logger.Error("              地址:" + imgPath)
			return
		}
		if ext == "jpeg" {
			ext = "jpg"
		}
		imgPath = imgPath + "." + ext
	}

	file, err := os.Create(imgPath)
	if err != nil {
		logger.Error("--------图片创建失败:" + imgSrc)
		logger.Error("              地址:" + imgPath)
		return
	}
	_, err = io.Copy(file, bytes.NewReader(robots))
	if err != nil {
		logger.Error("--------图片保存失败:" + imgSrc)
		logger.Error("              地址:" + imgPath)
		return
	}
	logger.Info("--------图片保存成功:" + imgSrc)
	logger.Info("              地址:" + imgPath)
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
	for i := 1; i < 8; i++ {
		getList(i)
	}
	logger.Info("请求完成,一小时后重试")
	time.Sleep(60 * 60 * time.Second)
	main()
}

func main1() {

	imgPath := "d:\\111\\"
	imgURL := "https://tva1.sinaimg.cn/large/007asALTgy1gl8ljm8thpj30mj0zttio.jpg"

	fileName := path.Base(imgURL)

	res, err := http.Get(imgURL)
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
	saveImg("https://tva1.sinaimg.cn/large/007qbgWbgy1geaz4l0mqpj30m80tm4ea.jpg", "d:/111/007qbgWbgy1geaz4l0mqpj30m80tm4ea.jpg")
}
