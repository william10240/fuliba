# 福利吧(https://fuliba2020.net) 福利汇总图片下载器(Golang版)

[Python 按本](https://github.com/williamyan1024/fuliimg_py), [NodeJs 版本](https://github.com/williamyan1024/fuliimg_js)

## 说明
- ~~保存2019年来福利汇总第二页的图片~~
- 福吧只显示最新7页的内容,所以只能保存最新的
- 每小时自动下载一遍,省事省心省力
- 如果网络不好,等网络好的时候会自动下载


## 自定义 图片存放路径
```
// app.go
// 定义 图片目录
var IMG_PATH = path.Join(filepath.Dir(APP_PATH), "fuliimages2")
```
