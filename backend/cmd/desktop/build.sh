#!/usr/bin/env bash
set -euo pipefail

# ACPone 桌面应用构建脚本
# 支持 macOS (Universal Binary), Windows, Linux

# Tips:
# macOS: brew install sips (built-in)
# Windows cross-compile: brew install mingw-w64
# Windows icon: go install github.com/akavel/rsrc@latest

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
WEB_DIR="$(cd "$ROOT_DIR/../web" && pwd)"

APP_NAME="ACPone"
IDENTIFIER="com.anthropic.acpone"
VERSION="1.0.0"
OUTPUT_DIR="$ROOT_DIR/dist"
ICON_DIR="$SCRIPT_DIR/icon"

# 颜色
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_fatal() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# 检查命令是否存在
check_command() {
    local cmd=$1
    local hint=${2:-""}
    if ! command -v "$cmd" &> /dev/null; then
        if [ -n "$hint" ]; then
            log_fatal "$cmd 未安装。$hint"
        else
            log_fatal "$cmd 未安装"
        fi
    fi
}

# 构建前端
build_web() {
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

# ==================== macOS ====================

build_mac_icon() {
    log_info "生成 macOS 图标..."

    if [ ! -f "$ICON_DIR/logo.png" ]; then
        log_warn "未找到 $ICON_DIR/logo.png，跳过图标生成"
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

    log_info "macOS 图标生成完成"
}

generate_info_plist() {
    cat > "$OUTPUT_DIR/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
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

build_mac_binary() {
    local arch=$1
    local output=$2

    log_info "构建 macOS $arch 二进制..."

    cd "$SCRIPT_DIR"

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

    [ -f "$output" ] || log_fatal "构建失败: $output"
    log_info "macOS $arch 二进制构建完成"
}

build_mac_universal() {
    local name="${APP_NAME}-mac-universal"

    log_info "构建 macOS Universal Binary..."

    mkdir -p "$OUTPUT_DIR"

    local app_dir="$OUTPUT_DIR/${APP_NAME}.app"
    rm -rf "$app_dir"
    mkdir -p "$app_dir/Contents/MacOS"
    mkdir -p "$app_dir/Contents/Resources"

    generate_info_plist
    cp "$OUTPUT_DIR/Info.plist" "$app_dir/Contents/Info.plist"

    [ -f "$OUTPUT_DIR/icon.icns" ] && cp "$OUTPUT_DIR/icon.icns" "$app_dir/Contents/Resources/icon.icns"

    local tmp_arm64="$OUTPUT_DIR/acpone-arm64"
    local tmp_amd64="$OUTPUT_DIR/acpone-amd64"

    build_mac_binary "arm64" "$tmp_arm64"
    build_mac_binary "amd64" "$tmp_amd64"

    log_info "合并为 Universal Binary..."
    lipo -create -output "$app_dir/Contents/MacOS/acpone" "$tmp_arm64" "$tmp_amd64"

    file "$app_dir/Contents/MacOS/acpone"
    lipo -info "$app_dir/Contents/MacOS/acpone"

    rm -f "$tmp_arm64" "$tmp_amd64"

    (cd "$OUTPUT_DIR" && zip -r "${name}.zip" "${APP_NAME}.app" 1>/dev/null)
    rm -rf "$app_dir"

    log_info "完成: $OUTPUT_DIR/${name}.zip"
}

build_mac() {
    local arch=$1
    local name="${APP_NAME}-mac-${arch}"

    log_info "构建 macOS $arch..."

    mkdir -p "$OUTPUT_DIR"

    local app_dir="$OUTPUT_DIR/${APP_NAME}.app"
    rm -rf "$app_dir"
    mkdir -p "$app_dir/Contents/MacOS"
    mkdir -p "$app_dir/Contents/Resources"

    generate_info_plist
    cp "$OUTPUT_DIR/Info.plist" "$app_dir/Contents/Info.plist"

    [ -f "$OUTPUT_DIR/icon.icns" ] && cp "$OUTPUT_DIR/icon.icns" "$app_dir/Contents/Resources/icon.icns"

    build_mac_binary "$arch" "$app_dir/Contents/MacOS/acpone"

    (cd "$OUTPUT_DIR" && zip -r "${name}.zip" "${APP_NAME}.app" 1>/dev/null)
    rm -rf "$app_dir"

    log_info "完成: $OUTPUT_DIR/${name}.zip"
}

# ==================== Windows ====================

build_windows() {
    local arch=${1:-amd64}
    local name="${APP_NAME}-windows-${arch}"

    log_info "构建 Windows $arch..."

    mkdir -p "$OUTPUT_DIR"
    cd "$SCRIPT_DIR"

    # 检查交叉编译器
    local cc=""
    case "$arch" in
        amd64)
            cc="x86_64-w64-mingw32-gcc"
            ;;
        386)
            cc="i686-w64-mingw32-gcc"
            ;;
        *)
            log_fatal "不支持的 Windows 架构: $arch"
            ;;
    esac

    if ! command -v "$cc" &> /dev/null; then
        log_fatal "$cc 未安装。请运行: brew install mingw-w64"
    fi

    # 生成 Windows 资源文件 (图标)
    local syso_file=""
    if [ -f "$ICON_DIR/logo.ico" ]; then
        if command -v rsrc &> /dev/null; then
            log_info "嵌入 Windows 图标..."
            rsrc -ico "$ICON_DIR/logo.ico" -o "$SCRIPT_DIR/rsrc_windows_${arch}.syso"
            syso_file="$SCRIPT_DIR/rsrc_windows_${arch}.syso"
        else
            log_warn "rsrc 未安装，跳过图标嵌入。安装: go install github.com/akavel/rsrc@latest"
        fi
    else
        log_warn "未找到 $ICON_DIR/logo.ico，跳过图标嵌入"
    fi

    # 构建
    log_info "编译 Windows $arch 二进制..."
    env GOOS=windows GOARCH=$arch CGO_ENABLED=1 CC=$cc \
        go build -ldflags "-H=windowsgui" -o "$OUTPUT_DIR/${APP_NAME}.exe" .

    [ -f "$OUTPUT_DIR/${APP_NAME}.exe" ] || log_fatal "构建失败"

    # 清理 syso 文件
    [ -n "$syso_file" ] && rm -f "$syso_file"

    # 打包
    (cd "$OUTPUT_DIR" && zip -r "${name}.zip" "${APP_NAME}.exe" 1>/dev/null)
    rm -f "$OUTPUT_DIR/${APP_NAME}.exe"

    log_info "完成: $OUTPUT_DIR/${name}.zip"
}

# ==================== Linux ====================

build_linux() {
    local arch=${1:-amd64}
    local name="${APP_NAME}-linux-${arch}"

    log_info "构建 Linux $arch..."

    mkdir -p "$OUTPUT_DIR"
    cd "$SCRIPT_DIR"

    # Linux 构建需要在 Linux 上进行（systray 需要 GTK）
    # 或者使用 Docker 进行交叉编译
    local current_os=$(uname -s)
    if [ "$current_os" != "Linux" ]; then
        log_warn "Linux 构建建议在 Linux 系统或 Docker 中进行"
        log_warn "尝试交叉编译（可能失败，因为 systray 需要 GTK）..."
    fi

    # 设置交叉编译器（如果有）
    local cc="gcc"
    local build_env="GOOS=linux GOARCH=$arch CGO_ENABLED=1"

    case "$arch" in
        amd64)
            if command -v x86_64-linux-gnu-gcc &> /dev/null; then
                cc="x86_64-linux-gnu-gcc"
            elif command -v x86_64-linux-musl-gcc &> /dev/null; then
                cc="x86_64-linux-musl-gcc"
            fi
            ;;
        arm64)
            if command -v aarch64-linux-gnu-gcc &> /dev/null; then
                cc="aarch64-linux-gnu-gcc"
            elif command -v aarch64-linux-musl-gcc &> /dev/null; then
                cc="aarch64-linux-musl-gcc"
            fi
            ;;
    esac

    build_env="$build_env CC=$cc"

    log_info "编译 Linux $arch 二进制..."
    if ! env $build_env go build -o "$OUTPUT_DIR/${APP_NAME}-linux-${arch}" .; then
        log_error "Linux 交叉编译失败。请在 Linux 系统或使用 GitHub Actions 构建。"
        return 1
    fi

    # 打包
    (cd "$OUTPUT_DIR" && zip -r "${name}.zip" "${APP_NAME}-linux-${arch}" 1>/dev/null)
    rm -f "$OUTPUT_DIR/${APP_NAME}-linux-${arch}"

    log_info "完成: $OUTPUT_DIR/${name}.zip"
}

# ==================== 清理 ====================

clean() {
    log_info "清理构建产物..."
    rm -rf "$OUTPUT_DIR"/*.zip
    rm -rf "$OUTPUT_DIR"/*.app
    rm -rf "$OUTPUT_DIR"/*.exe
    rm -rf "$OUTPUT_DIR"/*.icns
    rm -rf "$OUTPUT_DIR"/Info.plist
    rm -f "$SCRIPT_DIR"/*.syso
    log_info "清理完成"
}

# ==================== 帮助 ====================

show_help() {
    cat << EOF
ACPone 桌面应用构建脚本

用法:
    $0 [平台]

平台选项:
    mac-universal   macOS 通用包 (Intel + Apple Silicon) [默认]
    mac-arm64       macOS Apple Silicon
    mac-amd64       macOS Intel
    windows         Windows amd64
    windows-386     Windows 32位
    linux           Linux amd64
    linux-arm64     Linux ARM64
    all             构建所有平台 (需要对应工具链)
    clean           清理构建产物

示例:
    $0                      # 构建 macOS 通用包
    $0 mac-universal        # 构建 macOS 通用包
    $0 windows              # 构建 Windows 版本
    $0 linux                # 构建 Linux 版本
    $0 clean                # 清理

环境要求:
    macOS:   Xcode Command Line Tools (sips, iconutil, lipo)
    Windows: mingw-w64 (brew install mingw-w64)
             rsrc (go install github.com/akavel/rsrc@latest)
    Linux:   建议在 Linux 系统或 Docker 中构建

EOF
}

# ==================== 主函数 ====================

main() {
    local platform=${1:-mac-universal}

    case "$platform" in
        -h|--help|help)
            show_help
            exit 0
            ;;
        clean)
            clean
            exit 0
            ;;
    esac

    log_info "开始构建 $APP_NAME v$VERSION"
    log_info "平台: $platform"

    build_web

    case "$platform" in
        mac-universal|universal|mac)
            build_mac_icon
            build_mac_universal
            ;;
        mac-arm64|m1|arm64)
            build_mac_icon
            build_mac arm64
            ;;
        mac-amd64|intel)
            build_mac_icon
            build_mac amd64
            ;;
        windows|win|windows-amd64)
            build_windows amd64
            ;;
        windows-386|win32)
            build_windows 386
            ;;
        linux|linux-amd64)
            build_linux amd64
            ;;
        linux-arm64)
            build_linux arm64
            ;;
        all)
            build_mac_icon
            build_mac_universal
            build_windows amd64 || log_warn "Windows 构建失败"
            build_linux amd64 || log_warn "Linux 构建失败"
            ;;
        *)
            log_warn "未知平台: $platform"
            show_help
            exit 1
            ;;
    esac

    rm -f "$OUTPUT_DIR/Info.plist"

    log_info "构建完成!"
    ls -la "$OUTPUT_DIR"/*.zip 2>/dev/null || true
}

main "$@"
