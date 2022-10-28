go build -ldflags "-s -w" .
@chcp 65001
rem После upx с вероятностью, близкой к 100% агрится Касперский, поэтому не жмём
rem upx --best --lzma --overlay=strip *.exe
@chcp 866