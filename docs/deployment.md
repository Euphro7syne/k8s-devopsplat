# 部署

阶段 0 提供 `deploy/k3s/` 最小清单：Namespace、ServiceAccount/RBAC、SQLite PVC、ops-server、ops-web、Ingress。

默认部署到 `ops-platform` namespace。镜像标签由 Makefile 的 `SERVER_IMAGE`、`WEB_IMAGE` 控制。

阶段 1 调试建议先使用 demo workload，不直接接生产业务 Pod：

```bash
kubectl apply -f test/integration/fixtures/demo-workload.yaml
```

验证资源列表、Pod 日志、删除 Pod、Deployment 扩缩容和重启都通过后，再用只读权限接入真实业务 namespace。
