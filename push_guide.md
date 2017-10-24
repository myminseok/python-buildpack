
본 buildpack을 이용하면,  [Plotly Python library](https://plot.ly/), [Dash interactive Python framework](https://plot.ly/dash) 기반으로 작성된 python script를  웹기반 publishing하는 것을 자동화 할 수 있습니다.
어플리케이션은 [Pivotal Cloud Foundry](https://pivotal.io/platform)상에 자동으로 배포가 되며, 어플리케이션 구동에 필요한 web application framework, OS library가 자동으로 구성됩니다.
이때, 어플리케이션 마다 필요한 실행환경정보는 python script개발자가 아래의 파일로 제공하게 됩니다.

##  python script 실행환경 구성을 위한 파일

1. app.py : (필수)  https://plot.ly/dash/gallery 참조.
2. requirements.txt :(필수)
3. Procfile  : (필수)
4. manifest.yml: (필수)

## 작성 예시

### app.py
작성은  https://plot.ly/dash/gallery 참조.

```
... 중략

from flask import Flask
server = Flask('my app')
app = dash.Dash('GS Bond II Portfolio', server=server,
                url_base_pathname='/', csrf_protect=False)
app.scripts.config.serve_locally = True

... 중략

if __name__ == '__main__':
    app.server.run(host="0.0.0.0", port="9000")
```


### requirements.txt
어플리케이션에서 사용하는 dependency library목록을 기술합니다. 이때 index-url에 다운로드 받을 pypi repo를 지정합니다. default는 https://pypi.python.org/simple입니다.
http protocol일 경우 trusted-host에 지정합니다.
```
--index-url=http://pypi.domain.local/simple
--trusted-host pypi.domain.local
#--extra-index-url=http://pypi.domain.local2/simple
#--trusted-host pypi.domain.local2
numpy==1.12.0
pandas==0.19.2
pandas-datareader==0.3.0.post0
plotly==2.0.10
```

### Procfile
어플리케이션을 실행하는 스크립트로, 대부부분 변경없이 아래내용 그대로 사용가능합니다.

```
web: gunicorn app:server --timeout 300
```


### manifest.yml
```
---
applications:
- name: gold
  host: gold
  buildpack: python_buildpack
  memory: 2GB 
  disk: 4GB
  instances: 1
  timeout: 180
```
- name: (필수) PCF상에서 관리를 위한 어플리케이션 이름
- host: (선택, default는 name항목의 값) apps.pcf.sec.com 도메인의 하위이름으로 사용할 이름 지정.




## 참조
- PCF상에 python 어플리케이션 배포 가이드https://docs.run.pivotal.io/buildpacks/python/index.html
- 샘플참조: https://github.com/myminseok/dash-goldman-sachs-report-demo



