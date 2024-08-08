# Kubackup

<p align="center"><a href="https://kubackup.cn" target="_blank"><img src="https://kubackup.cn/img/kubackup-bar.png" alt="Kubackup" width="300" /></a></p>
<p align="center"><b>A Simple, Open-Source, and Modern Web UI for Restic</b></p>
<p align="center">
  <a href="https://github.com/kubackup/kubackup" target="_blank"><img alt="GitHub Repo stars" src="https://img.shields.io/github/stars/kubackup/kubackup?style=flat&logo=github"></a>
  <a href="https://gitee.com/kubackup/kubackup" target="_blank" style="padding-top: 5px"><img alt="Gitee Repo stars" src="https://gitee.com/kubackup/kubackup/badge/star.svg?theme=dark"></a>
  <a href="https://hub.docker.com/r/kubackup/kubackup"><img src="https://img.shields.io/docker/pulls/kubackup/kubackup" alt="Docker Pulls"/></a>
</p>
<p align="center">
  <a href="https://www.gnu.org/licenses/gpl-3.0.html" target="_blank"><img src="https://shields.io/github/license/kubackup/kubackup?color=%231890FF" alt="License: GPL v3"></a>
  <a href="https://github.com/kubackup/kubackup/releases"><img src="https://img.shields.io/github/v/release/kubackup/kubackup" alt="GitHub release"></a>
  <a href="https://app.codacy.com/gh/kubackup/kubackup/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade"><img src="https://app.codacy.com/project/badge/Grade/8e1a63fabe8b441cb31d5a70bd0291be" alt="Codacy"/></a>
  <a target="_blank" href="https://github.com/restic/restic/tree/v0.16.5"><img src="https://img.shields.io/badge/resitc-v0.16.5-red" alt="codacy"/></a>
</p>

[中文](README.md)

## Overview

[Kubackup](https://kubackup.cn) is a file backup system built on top of [Restic](https://github.com/restic/restic),
offering not only its powerful capabilities but also enhanced functionality and usability for a comprehensive data
protection experience. Kubackup retains Restic's core strengths, like high-speed backups, data security, and
reliability, while introducing the following features:

- **Simplicity**: User-friendly web interface with OTP authentication and account login, making it intuitive and easy
  for both beginners and experienced users to create, manage, and monitor backup tasks.
- **Efficiency**: Based on Restic's incremental backup technology, Kubackup only backs up data changed since the last
  backup, saving storage space while maintaining speed.
- **Speed**: Written in Go, it boosts backup efficiency, with backup speed limited only by network or disk bandwidth.
- **Security**: Data is encrypted during transmission and storage, ensuring confidentiality and integrity, while hash
  checks guarantee consistency.
- **Compatibility**: Supports importing existing Restic repositories.
- **Flexibility**: Extends support to Tencent COS, Alibaba OOS, and Huawei OBS as backend storage options, catering to
  Chinese users.

## Installation Documentation

[https://kubackup.cn/installation/online/](https://kubackup.cn/installation/online/)


## Demo

[https://demo.kubackup.cn](https://demo.kubackup.cn)  
username：```admin  ```  
password：```8aqxUYPt```

## Preview

<p align="center">
    <img style="box-shadow: 0 0 10px rgba(0,0,0,0.5);border-radius: 5px;" src="https://kubackup.cn/img/index.png" alt="Index"/>
</p>

## GitHub Stargazers over time

[![Stargazers over time](https://starchart.cc/kubackup/kubackup.svg?variant=light)](https://starchart.cc/kubackup/kubackup)

## Dependencies

- [Restic](https://github.com/restic/restic)
- [Cobra](https://github.com/spf13/cobra)
- [Iris](https://github.com/kataras/iris)
- [Vue](https://github.com/vuejs/vue)

## Disclaimer

**Users of this project should understand and accept the following risks:**

- The project may contain errors or undiscovered flaws, which could result in data loss, system failures, or other
  unforeseen issues. You are responsible for evaluating whether the software suits your requirements and implementing
  appropriate precautions, such as backing up your data and testing the software in an isolated environment, among
  others.
- The project may not be suitable for your specific requirements, and it is your responsibility to evaluate whether the
  project meets your requirements.
- The project might depend on third-party libraries or services, and the availability and stability of these
  dependencies are outside the control of the project author.
- The project author does not provide technical support, and you may need to resolve any encountered issues on your own
  or seek assistance from the community.

**Important Notice:** Before deploying the project into a production environment, it is strongly recommended that you
thoroughly test and validate it to ensure it aligns with your security and performance standards.
