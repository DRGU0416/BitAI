taskkill //F //FI "ImageName eq webui-*"
cp -f build/webui-* ../TaskService/
cd ../TaskService
start webui-qianyi.exe -WindowStyle Hidden
start webui-train.exe -WindowStyle Hidden
start webui-check.exe -WindowStyle Hidden
start webui-photohr.exe -WindowStyle Hidden