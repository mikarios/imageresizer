# image resizer
Microservice used to resize images. Supports connection to rabbitMQ to get jobs to execute and images within each job are processed concurrently with a set number of workers. Also provides an API to add jobs instead of using rabbitMQ.

----

<p align="center">
<a style="text-decoration: none" href="go.mod">
    <img src="https://img.shields.io/github/go-mod/go-version/mikarios/imageresizer?style=plastic" alt="Go version">
</a>


[//]: # (<a href="https://codecov.io/gh/mikarios/imageresizer" style="text-decoration: none">)

[//]: # (    <img src="https://img.shields.io/codecov/c/github/mikarios/imageresizer?label=codecov&style=plastic" alt="code coverage"/>)

[//]: # (</a>)

<a style="text-decoration: none" href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=plastic" alt="License: MIT">
</a>

<br />

<a style="text-decoration: none" href="https://github.com/mikarios/imageresizer/stargazers">
    <img src="https://img.shields.io/github/stars/mikarios/imageresizer.svg?style=plastic" alt="Stars">
</a>

<a style="text-decoration: none" href="https://github.com/mikarios/imageresizer/fork">
    <img src="https://img.shields.io/github/forks/mikarios/imageresizer.svg?style=plastic" alt="Forks">
</a>

<a style="text-decoration: none" href="https://github.com/mikarios/imageresizer/issues">
    <img src="https://img.shields.io/github/issues/mikarios/imageresizer.svg?style=plastic" alt="Issues">
</a>
</p>