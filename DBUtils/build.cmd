go build -o DBUtils.exe -ldflags "-s -w" .

rem После upx с вероятностью, близкой к 100% агрится Касперский, поэтому не жмём
rem upx --best --lzma --overlay=strip *.exe