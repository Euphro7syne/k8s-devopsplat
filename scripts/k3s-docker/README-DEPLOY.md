# ops-platform k3s Docker Runtime 验证包

适用于 k3s 使用 Docker runtime 的服务器。脚本不依赖 Python，服务器只需要常见的 `bash`、`awk`、`docker`、`kubectl`。

```bash
tar -xzf ops-platform-k3s-docker-*.tar.gz
cd ops-platform-k3s-docker-*

./prepare-config.sh
./build-images.sh
./deploy.sh
./apply-demo.sh

kubectl -n ops-platform port-forward svc/ops-web 18080:80
```

浏览器访问：

```text
http://127.0.0.1:18080
```

先用 `demo-app` namespace 验证资源列表、Pod 日志、Deployment 扩缩容/重启、Pod 删除和操作审计。真实业务 namespace 首次只做查看和日志验证。

如果包内存在 `images/ops-server.tar` 和 `images/ops-web.tar`，也可以执行 `./load-images.sh` 直接导入镜像；否则使用 `./build-images.sh` 在服务器 Docker 上构建。
