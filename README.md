# externalC2Client
External C2 Client 免杀效果自测。 
## 配置
1. Cobalt Strike listeners新建External C2 listener;
2. 填写端口;
![配置.png](https://github.com/Ed1s0nZ/externalC2Client/blob/main/%E9%85%8D%E7%BD%AE.png?raw=true)
3. 修改main.go中`var address = `127.0.0.1:8080``的IP和端口为Cobalt Strike External C2 listener的IP和端口；
## 编译
### Mac编译
1. brew install mingw-w64
2. GOOS=windows GOARCH=amd64 CC=/opt/homebrew/Cellar/mingw-w64/11.0.0/bin/x86_64-w64-mingw32-gcc CGO_ENABLED=1 go build -ldflags="-w -s" -ldflags="-H windowsgui" main.go
### Windows编译
1. go build -ldflags="-w -s" -ldflags="-H windowsgui" main.go
## 效果
![效果.png](https://github.com/Ed1s0nZ/externalC2Client/blob/main/%E6%95%88%E6%9E%9C.png?raw=true)
## 持续更新中，更新频率看star数量🐕。
# 声明：仅用于技术交流，请勿用于非法用途。


