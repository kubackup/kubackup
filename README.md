<p align="center"><a target="_blank" href="https://kubackup.cn"><img src="https://cos.kubackup.cn/img/kubackup-bar.png" alt="Kubackup" width="300" /></a></p>
<p align="center"><b>简单、开源、现代化的restic web UI</b></p>
<p align="center">
  <a target="_blank" href="https://github.com/kubackup/kubackup"><img alt="GitHub Repo stars" src="https://img.shields.io/github/stars/kubackup/kubackup?style=flat&logo=github"></a>
  <a target="_blank" style="padding-top: 5px" href="https://gitee.com/kubackup/kubackup"><img alt="Gitee Repo stars" src="https://gitee.com/kubackup/kubackup/badge/star.svg?theme=dark"></a>
  <a target="_blank" href="https://hub.docker.com/r/kubackup/kubackup"><img src="https://img.shields.io/docker/pulls/kubackup/kubackup" alt="docker pulls"/></a>
</p>
<p align="center">
  <a target="_blank" href="https://www.gnu.org/licenses/gpl-3.0.html"><img src="https://shields.io/github/license/kubackup/kubackup?color=%231890FF" alt="License: GPL v3"></a>
  <a target="_blank" href="https://github.com/kubackup/kubackup/releases"><img src="https://img.shields.io/github/v/release/kubackup/kubackup" alt="Github release"></a>
  <a target="_blank" href="https://app.codacy.com/gh/kubackup/kubackup/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade"><img src="https://app.codacy.com/project/badge/Grade/8e1a63fabe8b441cb31d5a70bd0291be" alt="codacy"/></a>
</p>

[English](README.en.md)

## 简介
    
[酷备份(kubackup)](https://kubackup.cn) 是一个基于[Restic](https://github.com/restic/restic)构建的文件备份系统，不但拥有 **Restic** 的强大能力，同时增加了丰富的功能和易用性，为用户提供更全面的数据保护体验。Kubackup保留了Restic的核心优势，如高速备份、数据安全性和可靠性，同时还引入了以下特性：

- **简单**：友好的Web界面，集成otp认证，账号密码登录，使得备份任务的创建、管理和监控变得更加直观和便捷，无论是新手还是经验丰富的用户都能轻松上手。
- **高效**：基于Restic的增量备份技术，Kubackup只备份自上次备份以来发生变化的数据，有效节省存储空间，同时保持备份速度。
- **快速**：使用 go 编程语言编写，充分提升备份效率，备份数据仅受网络或硬盘带宽的限制。
- **安全**：数据在传输和存储时全程加密，确保数据的机密性和完整性，同时利用哈希校验保证数据的一致性。
- **兼容**：支持导入您已有的Restic存储库。
- **多样性**：拓展**腾讯cos**、**阿里oos**、**华为obs**作为后端存储仓库，方便国内用户使用。

## 安装文档

[https://kubackup.cn](https://kubackup.cn/installation/online/)

## 页面

<p align="center">
    <img style="box-shadow: 0 0 10px rgba(0,0,0,0.5);border-radius: 5px;" src="https://cos.kubackup.cn/img/index.png" alt="index"/>
</p>

## GitHub Star 趋势图

[![Stargazers over time](https://starchart.cc/kubackup/kubackup.svg?variant=light)](https://starchart.cc/kubackup/kubackup)


## 主要用到的开源项目

* [Restic](https://github.com/restic/restic)
* [Cobra](https://github.com/spf13/cobra)
* [Iris](https://github.com/kataras/iris)
* [Vue](https://github.com/vuejs/vue)


## 免责声明

**使用本项目的用户应当理解并接受以下风险：**
- 项目可能包含错误或未发现的缺陷，可能导致数据丢失、系统崩溃或其他不可预见的问题，您应当自行评估软件是否适合您的需求，并采取适当的预防措施，包括但不限于备份数据、测试软件在隔离环境中。
- 项目可能不适合您的特定需求，您有责任评估项目是否满足您的要求。
- 项目可能依赖于第三方库或服务，这些依赖项的可用性和稳定性不在项目作者的控制范围内。
- 项目作者不提供技术支持，您可能需要自行解决遇到的问题或寻求社区支持。

**重要提示**：在将项目用于生产环境之前，强烈建议您进行充分的测试和验证，确保其符合您的安全和性能标准。
