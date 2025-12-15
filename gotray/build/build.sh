#!/usr/bin/env bash
set -e

# GoTray 通用构建脚本
# 使用方式: ./build.sh -n AppName -i identifier -v version [-p platform]

# 默认值
APP_NAME=""
IDENTIFIER=""
VERSION="1.0.0"
PLATFORM="all"
ICON_DIR="icon"
BUILD_DIR="build"
GO_BUILD_FLAGS=""
OUTPUT_DIR="dist"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

usage() {
    cat << EOF
GoTray 构建脚本

使用方式:
    $0 -n <app_name> -i <identifier> [options]

必需参数:
    -n  应用名称 (例如: MyApp)
    -i  应用标识符 (例如: com.example.myapp)

可选参数:
    -v  版本号 (默认: 1.0.0)
    -p  目标平台: mac, mac-arm64, win, linux, all (默认: all)
    -f  Go 构建标志 (例如: "-tags with_feature")
    -o  输出目录 (默认: dist)
    -h  显示帮助

示例:
    $0 -n MyApp -i com.example.myapp -v 1.0.0 -p mac
    $0 -n MyApp -i com.example.myapp -p all -f "-tags with_feature"
EOF
    exit 1
}

# 解析参数
while getopts "n:i:v:p:f:o:h" opt; do
    case $opt in
        n) APP_NAME="$OPTARG" ;;
        i) IDENTIFIER="$OPTARG" ;;
        v) VERSION="$OPTARG" ;;
        p) PLATFORM="$OPTARG" ;;
        f) GO_BUILD_FLAGS="$OPTARG" ;;
        o) OUTPUT_DIR="$OPTARG" ;;
        h) usage ;;
        *) usage ;;
    esac
done

# 验证必需参数
[ -z "$APP_NAME" ] && log_error "缺少应用名称 (-n)"
[ -z "$IDENTIFIER" ] && log_error "缺少应用标识符 (-i)"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 构建 macOS 图标
build_mac_icon() {
    log_info "生成 macOS 图标..."

    if [ ! -f "$ICON_DIR/logo.png" ]; then
        log_warn "未找到 $ICON_DIR/logo.png, 跳过图标生成"
        return
    fi

    rm -rf icons.iconset
    mkdir icons.iconset

    sips -z 16 16     "$ICON_DIR/logo.png" --out icons.iconset/icon_16x16.png      2>/dev/null
    sips -z 32 32     "$ICON_DIR/logo.png" --out icons.iconset/icon_16x16@2x.png   2>/dev/null
    sips -z 32 32     "$ICON_DIR/logo.png" --out icons.iconset/icon_32x32.png      2>/dev/null
    sips -z 64 64     "$ICON_DIR/logo.png" --out icons.iconset/icon_32x32@2x.png   2>/dev/null
    sips -z 128 128   "$ICON_DIR/logo.png" --out icons.iconset/icon_128x128.png    2>/dev/null
    sips -z 256 256   "$ICON_DIR/logo.png" --out icons.iconset/icon_128x128@2x.png 2>/dev/null
    sips -z 256 256   "$ICON_DIR/logo.png" --out icons.iconset/icon_256x256.png    2>/dev/null
    sips -z 512 512   "$ICON_DIR/logo.png" --out icons.iconset/icon_256x256@2x.png 2>/dev/null
    sips -z 512 512   "$ICON_DIR/logo.png" --out icons.iconset/icon_512x512.png    2>/dev/null
    sips -z 1024 1024 "$ICON_DIR/logo.png" --out icons.iconset/icon_512x512@2x.png 2>/dev/null

    iconutil -c icns icons.iconset -o "$OUTPUT_DIR/icon.icns"
    rm -rf icons.iconset

    log_info "图标生成完成"
}

# 生成 Info.plist
generate_info_plist() {
    local executable=$(echo "$APP_NAME" | tr '[:upper:]' '[:lower:]')

    cat > "$OUTPUT_DIR/Info.plist" << EOF
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleName</key><string>$APP_NAME</string>
	<key>CFBundleExecutable</key><string>$executable</string>
	<key>CFBundleIdentifier</key><string>$IDENTIFIER</string>
	<key>CFBundleVersion</key><string>$VERSION</string>
	<key>CFBundleGetInfoString</key><string>Built with GoTray</string>
	<key>CFBundleShortVersionString</key><string>$VERSION</string>
	<key>CFBundleIconFile</key><string>icon.icns</string>
	<key>LSMinimumSystemVersion</key><string>10.13.0</string>
	<key>NSHighResolutionCapable</key><string>true</string>
	<key>NSAppTransportSecurity</key><dict></dict>
	<key>LSUIElement</key><string>1</string>
</dict>
</plist>
EOF
}

# 构建 macOS 应用
build_mac() {
    local arch=$1
    local suffix=${2:-}
    local name="${APP_NAME}-mac-${arch}${suffix}"
    local executable=$(echo "$APP_NAME" | tr '[:upper:]' '[:lower:]')

    log_info "构建 macOS $arch$suffix..."

    # 创建 .app 目录结构
    local app_dir="$OUTPUT_DIR/${APP_NAME}.app"
    rm -rf "$app_dir"
    mkdir -p "$app_dir/Contents/MacOS"
    mkdir -p "$app_dir/Contents/Resources"

    # 复制 Info.plist
    generate_info_plist
    cp "$OUTPUT_DIR/Info.plist" "$app_dir/Contents/Info.plist"

    # 复制图标
    if [ -f "$OUTPUT_DIR/icon.icns" ]; then
        cp "$OUTPUT_DIR/icon.icns" "$app_dir/Contents/Resources/icon.icns"
    fi

    # 构建二进制
    local goamd64=""
    [ -n "$suffix" ] && goamd64="GOAMD64=$suffix"

    env GOOS=darwin GOARCH=$arch $goamd64 CGO_ENABLED=1 \
        go build $GO_BUILD_FLAGS -o "$app_dir/Contents/MacOS/$executable" .

    # 打包
    (cd "$OUTPUT_DIR" && zip -r "${name}.zip" "${APP_NAME}.app" 1>/dev/null)
    rm -rf "$app_dir"

    log_info "完成: $OUTPUT_DIR/${name}.zip"
}

# 构建 Windows 应用
build_win() {
    log_info "构建 Windows amd64..."

    local executable=$(echo "$APP_NAME" | tr '[:upper:]' '[:lower:]')

    # 生成 manifest 和资源文件
    if command -v rsrc &> /dev/null; then
        if [ -f "$ICON_DIR/logo.ico" ]; then
            # 生成 manifest
            cat > "$OUTPUT_DIR/app.manifest" << EOF
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
    <assemblyIdentity version="${VERSION}.0" processorArchitecture="*" name="$APP_NAME" type="win32"/>
    <dependency>
        <dependentAssembly>
            <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
        </dependentAssembly>
    </dependency>
</assembly>
EOF
            rsrc -manifest "$OUTPUT_DIR/app.manifest" -ico "$ICON_DIR/logo.ico" -o "${executable}.syso"
        fi
    fi

    # 构建 (需要 mingw-w64)
    if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
        env GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" \
            go build $GO_BUILD_FLAGS -ldflags -H=windowsgui -o "$OUTPUT_DIR/${APP_NAME}.exe" .

        (cd "$OUTPUT_DIR" && zip -r "${APP_NAME}-win-amd64.zip" "${APP_NAME}.exe" 1>/dev/null)
        rm -f "$OUTPUT_DIR/${APP_NAME}.exe"

        log_info "完成: $OUTPUT_DIR/${APP_NAME}-win-amd64.zip"
    else
        log_warn "未找到 mingw-w64, 跳过 Windows 构建"
    fi

    # 清理
    rm -f "${executable}.syso" "$OUTPUT_DIR/app.manifest"
}

# 构建 Linux 应用
build_linux() {
    log_info "构建 Linux amd64..."

    local executable=$(echo "$APP_NAME" | tr '[:upper:]' '[:lower:]')

    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
        go build $GO_BUILD_FLAGS -o "$OUTPUT_DIR/${executable}-linux-amd64" .

    (cd "$OUTPUT_DIR" && tar -czf "${APP_NAME}-linux-amd64.tar.gz" "${executable}-linux-amd64")
    rm -f "$OUTPUT_DIR/${executable}-linux-amd64"

    log_info "完成: $OUTPUT_DIR/${APP_NAME}-linux-amd64.tar.gz"
}

# 主构建流程
main() {
    log_info "开始构建 $APP_NAME v$VERSION"

    case "$PLATFORM" in
        mac)
            build_mac_icon
            build_mac amd64
            ;;
        mac-arm64|m1)
            build_mac_icon
            build_mac arm64
            ;;
        mac-v3)
            build_mac_icon
            build_mac amd64 v3
            ;;
        win)
            build_win
            ;;
        linux)
            build_linux
            ;;
        all)
            build_mac_icon
            build_mac amd64
            build_mac arm64
            build_win
            build_linux
            ;;
        *)
            log_error "未知平台: $PLATFORM"
            ;;
    esac

    # 清理临时文件
    rm -f "$OUTPUT_DIR/Info.plist" "$OUTPUT_DIR/icon.icns"

    log_info "构建完成!"
    ls -la "$OUTPUT_DIR"
}

main
