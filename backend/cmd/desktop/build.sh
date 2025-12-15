#!/usr/bin/env bash
set -e

# ACPone 桌面应用构建脚本

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
WEB_DIR="$(cd "$ROOT_DIR/../web" && pwd)"

APP_NAME="ACPone"
IDENTIFIER="com.anthropic.acpone"
VERSION="1.0.0"
OUTPUT_DIR="$ROOT_DIR/dist"
ICON_DIR="$SCRIPT_DIR/icon"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# 构建前端
build_web() {
    # 如果 dist 目录已存在且有内容，跳过构建 (CI 环境下已构建)
    if [ -f "$WEB_DIR/dist/index.html" ]; then
        log_info "前端已构建，跳过..."
        return
    fi

    log_info "构建前端..."
    cd "$WEB_DIR"
    npm install
    npm run build
    log_info "前端构建完成"
}

# 生成 macOS 图标
build_mac_icon() {
    log_info "生成 macOS 图标..."

    if [ ! -f "$ICON_DIR/logo.png" ]; then
        log_warn "未找到 $ICON_DIR/logo.png"
        return
    fi

    mkdir -p "$OUTPUT_DIR"
    local iconset="$OUTPUT_DIR/icons.iconset"

    rm -rf "$iconset"
    mkdir "$iconset"

    sips -z 16 16     "$ICON_DIR/logo.png" --out "$iconset/icon_16x16.png"      2>/dev/null
    sips -z 32 32     "$ICON_DIR/logo.png" --out "$iconset/icon_16x16@2x.png"   2>/dev/null
    sips -z 32 32     "$ICON_DIR/logo.png" --out "$iconset/icon_32x32.png"      2>/dev/null
    sips -z 64 64     "$ICON_DIR/logo.png" --out "$iconset/icon_32x32@2x.png"   2>/dev/null
    sips -z 128 128   "$ICON_DIR/logo.png" --out "$iconset/icon_128x128.png"    2>/dev/null
    sips -z 256 256   "$ICON_DIR/logo.png" --out "$iconset/icon_128x128@2x.png" 2>/dev/null
    sips -z 256 256   "$ICON_DIR/logo.png" --out "$iconset/icon_256x256.png"    2>/dev/null
    sips -z 512 512   "$ICON_DIR/logo.png" --out "$iconset/icon_256x256@2x.png" 2>/dev/null
    sips -z 512 512   "$ICON_DIR/logo.png" --out "$iconset/icon_512x512.png"    2>/dev/null
    sips -z 1024 1024 "$ICON_DIR/logo.png" --out "$iconset/icon_512x512@2x.png" 2>/dev/null

    iconutil -c icns "$iconset" -o "$OUTPUT_DIR/icon.icns"
    rm -rf "$iconset"

    log_info "图标生成完成"
}

# 生成 Info.plist
generate_info_plist() {
    cat > "$OUTPUT_DIR/Info.plist" << EOF
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>CFBundleName</key><string>$APP_NAME</string>
    <key>CFBundleExecutable</key><string>acpone</string>
    <key>CFBundleIdentifier</key><string>$IDENTIFIER</string>
    <key>CFBundleVersion</key><string>$VERSION</string>
    <key>CFBundleGetInfoString</key><string>ACP Gateway Chat Interface</string>
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


# 构建单架构二进制
build_binary() {
    local arch=$1
    local output=$2

    log_info "构建 $arch 二进制..."

    cd "$SCRIPT_DIR"

    # macOS clang 支持交叉编译，设置目标架构
    local cgo_cflags=""
    local cgo_ldflags=""
    if [ "$arch" = "amd64" ]; then
        cgo_cflags="-arch x86_64"
        cgo_ldflags="-arch x86_64"
    elif [ "$arch" = "arm64" ]; then
        cgo_cflags="-arch arm64"
        cgo_ldflags="-arch arm64"
    fi

    env GOOS=darwin GOARCH=$arch CGO_ENABLED=1 \
        CGO_CFLAGS="$cgo_cflags" CGO_LDFLAGS="$cgo_ldflags" \
        go build -o "$output" .

    if [ ! -f "$output" ]; then
        log_error "构建失败: $output"
    fi

    log_info "$arch 二进制构建完成"
}

# 构建 macOS 应用
build_mac() {
    local arch=$1
    local name="${APP_NAME}-mac-${arch}"

    log_info "构建 macOS $arch..."

    mkdir -p "$OUTPUT_DIR"

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
    build_binary "$arch" "$app_dir/Contents/MacOS/acpone"

    # 打包
    (cd "$OUTPUT_DIR" && zip -r "${name}.zip" "${APP_NAME}.app" 1>/dev/null)

    log_info "完成: $OUTPUT_DIR/${name}.zip"
}

# 构建 Universal Binary (同时支持 Intel 和 Apple Silicon)
build_mac_universal() {
    local name="${APP_NAME}-mac-universal"

    log_info "构建 macOS Universal Binary..."

    mkdir -p "$OUTPUT_DIR"

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

    # 构建两个架构的二进制
    local tmp_arm64="$OUTPUT_DIR/acpone-arm64"
    local tmp_amd64="$OUTPUT_DIR/acpone-amd64"

    build_binary "arm64" "$tmp_arm64"
    build_binary "amd64" "$tmp_amd64"

    # 使用 lipo 合并为 Universal Binary
    log_info "合并为 Universal Binary..."
    lipo -create -output "$app_dir/Contents/MacOS/acpone" "$tmp_arm64" "$tmp_amd64"

    # 验证
    file "$app_dir/Contents/MacOS/acpone"
    lipo -info "$app_dir/Contents/MacOS/acpone"

    # 清理临时文件
    rm -f "$tmp_arm64" "$tmp_amd64"

    # 打包
    (cd "$OUTPUT_DIR" && zip -r "${name}.zip" "${APP_NAME}.app" 1>/dev/null)

    log_info "完成: $OUTPUT_DIR/${name}.zip"
}

# 主函数
main() {
    local platform=${1:-mac-universal}

    log_info "开始构建 $APP_NAME v$VERSION"
    log_info "平台: $platform"

    # 构建前端
    build_web

    case "$platform" in
        mac-amd64|intel)
            build_mac_icon
            build_mac amd64
            ;;
        mac-arm64|m1|arm64)
            build_mac_icon
            build_mac arm64
            ;;
        mac-universal|universal|mac)
            build_mac_icon
            build_mac_universal
            ;;
        all)
            build_mac_icon
            build_mac_universal
            build_mac amd64
            build_mac arm64
            ;;
        *)
            log_warn "未知平台: $platform, 使用 mac-universal"
            build_mac_icon
            build_mac_universal
            ;;
    esac

    # 清理临时文件
    rm -f "$OUTPUT_DIR/Info.plist"

    log_info "构建完成!"
    ls -la "$OUTPUT_DIR"
}

main "$@"
