# XDSec 2026 招新系统后端

西电信安协会 2026 年招新网站后端，使用 Golang 构建

前端仓库在[这里](https://github.com/CopperKoi/XDSEC-Recruitment-System)！

相关文章：[西电信安协会招新系统 Golang 后端开发小记](https://www.xiaozonglin.cn/xidian-cybersecurity-association-recruitment-system-golang-backend-development-notes/)

## 主要功能

面向面试者：

- 提交简历和申请
- 提交任务报告
- 查看面试状态
- 查看面试公告

面向面试官：

- 发布、编辑面试公告
- 查看面试者的简历
- 设置面试者面试状态
- 给面试者布置任务并查看其提交的报告
- 在简历中留言（仅面试官可见）

面向管理员：

- 快速导出所有面试者的信息

## 部署

将`.env.example`中的内容填充修改好后重命名为`.env`，与编译产物放置于同一目录。

其中的`secretKey`没有用，可以考虑在本地修改`auth/jwt.go`中的`jwtSecret`值再编译。

## 接口文档

详见[API.md](https://github.com/CopperKoi/XDSEC-Recruitment-System/blob/main/docs/api.zh.md)
