# [reecemercer.dev](reecemercer.dev) backend

## This repository contains the source code for the backend server that provides various services to my personal website. It is hosted on a Heroku Dyno and is publicly accessible.

It is a successor to my previous backend (written in NodeJS). It simplifies things for me and also makes the actual API itself easier to use - some of the tasks one request achieves here took multiple separate requests via the previous API. Bundling them together made more sense from an ease of use as well as an ease of development standpoint.

And not that it matters with a personal website - but I also load tested and the server can handle thousands of concurrent requests with ease. Thanks Go :)

## **Services:**

- Live public GitHub repo summary
- Language distribution over public repos as summarised above
- Lists of photo collections from my AWS S3 instance
- Contents of any collection from the instance above

# Development instructions

```powershell
cd $env:GOPATH\src
git clone git@github.com:Reeceeboii/personal-website-backend.git
cd .\personal-website-backend
code .
go build
personal-website-backend.exe
```

## Adding new deps

```powershell
go get [<pkg@master> | <pkg@v1.1.1>]
```

## Before each release

```powershell
go mod tidy
```

- prunes pkg list and removes all unnecessary packages
