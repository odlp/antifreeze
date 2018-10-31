SET GOPATH=%CD%\go
SET CF_DIAL_TIMEOUT=15

SET PATH=C:\Go\bin;%PATH%
SET PATH=%GOPATH%\bin;%PATH%
SET PATH=C:\Program Files\GnuWin32\bin;%PATH%
SET PATH=%CD%;%PATH%

SET /p DOMAIN=<%CD%\bosh-lock\name
SET /p CF_INT_PASSWORD=<%CD%\cf-credentials\cf-password
SET CF_INT_API=https://api.%DOMAIN%

pushd %CD%\cf-cli-binaries
	7z e cf-cli-binaries.tgz
	7z x cf-cli-binaries.tar
	MOVE %CD%\cf-cli_winx64.exe ..\cf.exe
popd

go get -v -u github.com/onsi/ginkgo/ginkgo

SET RUN_ID=(openssl rand -hex 16)

cd %GOPATH%\src\code.cloudfoundry.org\cli
ginkgo.exe -r -nodes=16 -flakeAttempts=2 -slowSpecThreshold=60 -randomizeAllSpecs ./integration/shared/experimental ./integration/v6/experimental
