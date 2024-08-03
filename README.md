# nght

a gin web server for nginx http test ( nginx gin http test  or nginx http test)

本项目使用golang 和 gin 进行开发

使用golang开发的是因为go install 之后的bin文件可以直接运行,python和java都需要有依赖环境

使用gin开发是因为其很简单


## python版本

uvicorn nght:app --host 0.0.0.0 --port 8000 --reload
uvicorn nght:app --host 0.0.0.0 --port 8001 --reload

## todo

1. 动态创建path,动态销毁path
2. a api for create url path 
3. use golang fasthttp
