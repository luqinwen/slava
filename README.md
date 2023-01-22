# Slava 🎩
A High-performance、K-V Cloud Database.

## Reference Urls：
1. [github地址](https://github.com/luqinwen/slava)
2. [git操作参考](https://www.runoob.com/git/git-tutorial.html)
3. [小林coding](https://xiaolincoding.com/redis/)
4. [极客时间: Redis核心技术与实战](https://time.geekbang.org/column/intro/100056701)
5. [7-days http server](https://geektutu.com/post/gee-day1.html)

## PR规范
1. commit message 必须以todo编号: message 的样式做记录
分支type：
feature - 新功能 feature
fix - 修复 bug
docs - 文档注释
style - 代码格式(不影响代码运行的变动)
refactor - 重构、优化(既不增加新功能，也不是修复bug)
perf - 性能优化
test - 增加测试
chore - 构建过程或辅助工具的变动
revert - 回退
build - 打包

2. todo编号的命名规则是：分支type/版本号-***（三位数的编号），eg: feature:0-001
3. 多个commit需要squash后再进行提交
4. 合并分支commit和rebase结合使用
5. Commit 信息规范：
> 0-002:summary
>
> 1. fix the bug xxx
> 2. xxx
> 3. xxx
6. 提交分支 feature/v0、test/v0（不要提交到main分支，由我来做最后的合并）
7. 提交pr后需要拉一个1-1的会与我对齐你的进度以及讲解你的代码同时进行code review

## 关于提交 PR 的方法：
### Step1:
首先你需要 fork 本仓库到你自己的 github 仓库，点击右上角的 fork 按钮。
### Step2:
使用 git clone 命令将本仓库拷贝到你的本地文件，git clone 地址请点开项目上方的绿色 "code" 按钮查看。
### Step3:
在你的本地按照todo对代码进行修改、提交。
### Step4:
修改完后，是时候该上传你的改动到你 fork 来的远程仓库上了。你可以用 git bash，也可以使用 IDE 里的 git 来操作。对于 git 不熟的用户建议使用 IDE，IDE 也更方便写 commit 信息。
### Step5:
上传之后，点进你的仓库主页，会出现一个 "Contribute"，点击它，选择 "Open pull request"，选择好你仓库的分支和你想要在这里合并的分支后，点击 "Create pull request"，之后填写你的 PR 标题和正文内容，就成功提交一个 PR 。接下来等待我的approve/feedback。
### Step6 (optional):
记得检查修改自己的 GitHub Public profile 里的 Name 和 Public email，位置在右上角头像的 Settings 里，因为大多数情况下我们会使用 squash merge 来合并 PRs，此时 squash merge 后产生的新提交作者信息会使用这个 GH 信息。

## Todo
### [开发看板](https://gbvsqqoj6n.feishu.cn/docx/VtzXdoU7coNdLLxtHnmc9MkGnxf)
### V0（计划开发周期1.9～3.6）

| Todo codes   | Issues |Contributors|
| :----- | :-----  |:-----|
| docs:0-001|  1. Readme update  | [Qinwen](https://github.com/luqinwen)|
| feature:0-002|  1. TCP simple server | 
| feature:0-003|1. Well developed tcp server|
|feature:0-004|1. Echo handler|

### V1（Docker+K8S/Cloud K-V Database）
