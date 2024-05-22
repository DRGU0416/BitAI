rm build/webui-*
# go build -trimpath -o build/webui-api.exe main.go
# if [ $? -ne 0 ];then
# 	echo 'api服务编译失败'
# else
# 	echo 'api服务编译成功 ...'
# fi

cd task
cd qianyi
go build -trimpath -o ../../build/webui-qianyi.exe main.go
if [ $? -ne 0 ];then
	echo 'qianyi服务编译失败'
else
	echo 'qianyi服务编译成功 ...'
fi

cd ..
cd train
go build -trimpath -o ../../build/webui-train.exe main.go
if [ $? -ne 0 ];then
	echo 'train服务编译失败'
else
	echo 'train服务编译成功 ...'
fi

cd ..
cd check
go build -trimpath -o ../../build/webui-check.exe main.go
if [ $? -ne 0 ];then
	echo 'check服务编译失败'
else
	echo 'check服务编译成功 ...'
fi

cd ..
cd photohr
go build -trimpath -o ../../build/webui-photohr.exe main.go
if [ $? -ne 0 ];then
	echo '高清服务编译失败'
else
	echo '高清服务编译成功 ...'
fi