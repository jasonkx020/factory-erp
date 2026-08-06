# 加工厂 ERP · 员工 App（Flutter）

Android / iOS **唯一**员工现场端。与管理端共用 `/api/v1` 与 IAM；登录 `client_type=mobile`。

后端 API 用 Go 启动；Go 国内安装与代理的精简说明也见仓库根目录 [README.md](../README.md)。

## 环境要求

| 组件 | 说明 |
|------|------|
| Go | 与仓库根 `go.mod` 一致（当前 `1.26.x`），用于跑 `erp-api` |
| Flutter SDK | 建议单独目录，如 Windows `D:\flutter`、Linux `~/flutter` |
| Android Studio | 含 Android SDK、平台工具；JDK 17（可用 Studio 自带 JBR） |
| 设备 | 真机 USB 调试或模拟器；电脑与手机同一局域网 |
| API | 默认 `:18080`，前缀 `/api/v1` |

演示账号：`admin` / `admin123`

---

## Go 安装与国内代理

跑员工 App 前需本机可访问 API。国内安装后**务必**改 `GOPROXY`，否则 `go mod download` 很慢或失败。

### Windows

1. 从 [golang.google.cn/dl](https://golang.google.cn/dl/) 或 [go.dev/dl](https://go.dev/dl/) 下载 `.msi` 安装。
2. 新开 PowerShell：

```powershell
go version

# 安装后立刻改为国内模块代理（永久）
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
go env GOPROXY GOSUMDB
```

3. 启动 API（仓库根目录）：

```powershell
cd D:\workplace\ycwl-erp-master
go mod download
go run ./cmd/erp-api
```

健康检查：浏览器打开 `http://127.0.0.1:18080/api/v1/health`。

### Linux

```bash
# 示例版本按官网调整
curl -fsSL -o go.tgz https://golang.google.cn/dl/go1.26.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go.tgz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc
go version

go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn

cd /path/to/ycwl-erp-master
go mod download
go run ./cmd/erp-api
```

备选 `GOPROXY`：`https://proxy.golang.com.cn,direct`、`https://mirrors.aliyun.com/goproxy/,direct`。

---

## Flutter 安装与收尾配置

### Windows：安装 Flutter

1. 安装 **Git for Windows**、**Android Studio**（SDK Manager 勾选 Android SDK、平台工具、至少一个较新 Platform）。
2. 克隆 SDK（建议与工程**同一盘符**，避免 Kotlin 跨盘符问题）：

```powershell
cd D:\
git clone https://github.com/flutter/flutter.git -b stable
# 国内可改用镜像站 clone，或配置 git 代理后再拉
```

3. 将 `D:\flutter\bin` 加入用户 **PATH**（系统设置 → 环境变量），**新开**终端：

```powershell
flutter --version
```

4. **国内镜像（必配，永久）**：

```powershell
[Environment]::SetEnvironmentVariable("FLUTTER_STORAGE_BASE_URL", "https://storage.flutter-io.cn", "User")
[Environment]::SetEnvironmentVariable("PUB_HOSTED_URL", "https://pub.flutter-io.cn", "User")
# 可选：pub 缓存与工程同盘
[Environment]::SetEnvironmentVariable("PUB_CACHE", "D:\flutter\pub-cache", "User")
# 可选：Gradle 缓存集中存放
[Environment]::SetEnvironmentVariable("GRADLE_USER_HOME", "D:\flutter\gradle-cache", "User")
```

新开 PowerShell 后确认：

```powershell
echo $env:FLUTTER_STORAGE_BASE_URL
echo $env:PUB_HOSTED_URL
```

5. 接受 Android 许可并体检：

```powershell
flutter doctor --android-licenses
flutter doctor -v
```

按提示补齐：Android toolchain、设备/`chrome`（本项目不做 Flutter Web 亦可忽略 Web）。

### Linux：安装 Flutter

```bash
# 依赖（Debian/Ubuntu 示例）
sudo apt update
sudo apt install -y curl git unzip xz-utils zip libglu1-mesa

cd ~
git clone https://github.com/flutter/flutter.git -b stable
echo 'export PATH="$PATH:$HOME/flutter/bin"' >> ~/.bashrc

# 国内镜像（写入 shell 配置，长期生效）
cat >> ~/.bashrc <<'EOF'
export FLUTTER_STORAGE_BASE_URL=https://storage.flutter-io.cn
export PUB_HOSTED_URL=https://pub.flutter-io.cn
export PUB_CACHE=$HOME/flutter/pub-cache
EOF
source ~/.bashrc

flutter doctor --android-licenses
flutter doctor -v
```

Android SDK 可用 Android Studio 或命令行 `cmdline-tools`；设置 `ANDROID_HOME` / `ANDROID_SDK_ROOT` 指向 SDK 根目录。

### 收尾配置清单（装完必做）

| 步骤 | Windows | Linux |
|------|---------|-------|
| Go 代理 | `go env -w GOPROXY=https://goproxy.cn,direct` | 同左 |
| Flutter 存储/Pub 镜像 | User 环境变量见上 | `~/.bashrc` 见上 |
| `flutter doctor` 无红色阻塞项 | 是 | 是 |
| 真机 USB 调试 / 无线调试 | 开启开发者选项 | 同左 |
| 本仓库 Android 已预置 | 腾讯云 Gradle、阿里云 Maven、desugaring | 同左 |
| API 已启动且手机能访问电脑 IP | `go run ./cmd/erp-api` | 同左 |
| 防火墙放行 18080 | Windows 防火墙入站 | `ufw`/安全组按需 |

可选：Clash 等代理仅给浏览器用时，**不要**假设 Gradle/Java 自动走代理；优先靠 `FLUTTER_STORAGE_BASE_URL`，详见下文踩坑 §5。

### 运行本 App

```powershell
# Windows 示例；先确认 API 已监听 :18080
cd D:\workplace\ycwl-erp-master\mobile
flutter pub get
flutter devices
flutter run -d <设备ID> --dart-define=API_BASE=http://192.168.x.x:18080/api/v1
```

```bash
# Linux
cd /path/to/ycwl-erp-master/mobile
flutter pub get
flutter run -d <设备ID> --dart-define=API_BASE=http://192.168.x.x:18080/api/v1
```

查看完整编译过程：

```powershell
flutter run -d <设备ID> -v --dart-define=API_BASE=http://192.168.x.x:18080/api/v1
# 或
cd android
.\gradlew.bat assembleDebug --info --stacktrace   # Linux: ./gradlew
```

## 模块（一二三期已齐，可交付）

| 模块 | 能力 |
|------|------|
| 车间工作台 | 扫码报工、概览、派工接收、灵活派发、质检/废料/返修、图纸 |
| 工人报工 | 双扫、今日核对、联动领料过账 |
| 过磅收货 | 过磅/质检/出码推仓、采购任务 |
| 仓管作业 | 待入库、库存预警、箱码、盘点、扫码出入库 |
| 销售外勤 | 订单/询价/出厂、发货进度、报价试算、跟进 |
| 固定资产 | 查询、内部转移申请与确认 |
| 收款协同 | 收款预警处理、销售认款 |
| 资料中心 | 知识库/图纸/公告/学堂只读 |
| 我的 | 打卡、假勤、审批、工资提成、消息、日志/备忘录 |

验收：[`DELIVERY.md`](DELIVERY.md) · 冒烟：`go run ./cmd/mobile_delivery_smoke`（在仓库根目录）

## 不包含

- Flutter Web / uni-app / 客户自助前端

---

## Android 编译踩坑与解决办法（安装开发参考）

以下问题多见于 **Windows + 国内网络 + 代理**。本仓库已预置部分配置；新环境仍请按现象对照处理。

### 1. Gradle Wrapper 下载超时 / Connection timed out

**现象**：`java.net.ConnectException` / `SocksSocketImpl`，卡在下载 `services.gradle.org` 的 `gradle-*-all.zip`。

**原因**：官方源慢或不可达；本机开了 SOCKS/HTTP 代理但 Java 走不通。

**解决**：

1. 使用腾讯云发行包镜像（本仓库已改）：

   [`android/gradle/wrapper/gradle-wrapper.properties`](android/gradle/wrapper/gradle-wrapper.properties)

   ```properties
   distributionUrl=https\://mirrors.cloud.tencent.com/gradle/gradle-8.14-all.zip
   ```

2. 备选：`https://mirrors.huaweicloud.com/gradle/gradle-8.14-all.zip`

3. 若改过 `distributionUrl` 后卡住：不同 URL 对应不同缓存目录。删掉半截缓存后重试，或手动用浏览器/`curl` 下完整 zip（约 214MB）放到：

   `%GRADLE_USER_HOME%\wrapper\dists\gradle-8.14-all\<hash>\`

   解压出 `gradle-8.14\`，并生成空文件 `gradle-8.14-all.zip.ok`。

4. 代理：要么保证 Clash 等代理端口可用，要么临时关闭系统代理再编。

> Flutter 可能使用独立 `GRADLE_USER_HOME`（例如 `D:\flutter\gradle-cache`），以实际环境变量为准，不要只清 `~/.gradle`。

### 2. Maven 依赖 Connection reset / SSL misconfiguration

**现象**：无法从 `repo.maven.apache.org` / Google Maven 拉 `kotlin-gradle-plugin` 等。

**解决**：项目已优先阿里云仓库：

- [`android/settings.gradle.kts`](android/settings.gradle.kts)（`pluginManagement.repositories`）
- [`android/build.gradle.kts`](android/build.gradle.kts)（`allprojects.repositories`）

**不要**在 `%GRADLE_USER_HOME%\init.gradle` 里对 `settings.pluginManagement.repositories` 做 `clear()` 再塞镜像——会与 Flutter 的 `PREFER_SETTINGS` 冲突，报错类似：

```text
Build was configured to prefer settings repositories over project repositories
but repository 'maven' was added by settings file 'settings.gradle.kts'
```

### 3. `flutter_local_notifications` 要求 core library desugaring

**现象**：

```text
Dependency ':flutter_local_notifications' requires core library desugaring to be enabled for :app
```

**解决**（本仓库 [`android/app/build.gradle.kts`](android/app/build.gradle.kts) 已配置）：

```kotlin
compileOptions {
    isCoreLibraryDesugaringEnabled = true
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}
defaultConfig {
    multiDexEnabled = true
}
dependencies {
    coreLibraryDesugaring("com.android.tools:desugar_jdk_libs:2.1.4")
}
```

### 4. Kotlin 增量编译跨盘符失败

**现象**：

```text
this and base files have different roots:
E:\...\pub-cache\...\MessagesAsync.g.kt and D:\...\mobile\android
```

**原因**：工程与 `PUB_CACHE` / Flutter SDK 不在同一盘符时，Kotlin incremental cache 无法算相对路径。

**解决**（任选）：

1. 把 `PUB_CACHE`、Flutter SDK、工程放到**同一盘符**（推荐）：

   ```powershell
   [Environment]::SetEnvironmentVariable("PUB_CACHE", "D:\flutter\pub-cache", "User")
   ```

2. 或在 [`android/gradle.properties`](android/gradle.properties) 关闭增量：

   ```properties
   kotlin.incremental=false
   kotlin.incremental.android=false
   ```

3. 清理后重编：`flutter clean`，必要时删 `mobile/build`、`mobile/android/.gradle`（先结束占用中的 Gradle/Java 进程）。

### 5. 卡住下载 `storage.googleapis.com/.../arm64_v8a_debug-*.jar`

**现象**：日志停在 Gradle 下载 Flutter 引擎 AAR/JAR；**浏览器能下、Gradle 卡住**。

**原因**：浏览器走系统/Clash 代理；Gradle/Java 默认不走，直连 Google Storage 易挂起。

**解决**：

```powershell
$env:FLUTTER_STORAGE_BASE_URL = "https://storage.flutter-io.cn"
$env:PUB_HOSTED_URL = "https://pub.flutter-io.cn"
```

用 `-v` 确认地址变成 `storage.flutter-io.cn/download.flutter.io/...`。修改 User 环境变量后须**新开终端**。

可选：给 Gradle 显式代理（端口按本机 Clash 为准，常见 `7897`）：

```properties
# android/gradle.properties
systemProp.http.proxyHost=127.0.0.1
systemProp.http.proxyPort=7897
systemProp.https.proxyHost=127.0.0.1
systemProp.https.proxyPort=7897
```

代理未启动时不要写这些项，否则会更慢。

### 6. `Running Gradle task 'assembleDebug'...` 长时间无输出

**排查顺序**：

1. `flutter run -v` 看停在 Wrapper、Maven 还是 Flutter 引擎包
2. 任务管理器是否多个 `java`/`gradlew` 互相抢锁 → 结束旧进程后只保留一次编译
3. 检查 `%GRADLE_USER_HOME%\wrapper\dists\` 下是否有 `*.part` 且大小长期为 0
4. 首次编译会下数百 MB 依赖，国内镜像配好后通常数分钟内有进展

### 7. SDK XML version 警告

**现象**：`This version only understands SDK XML versions up to 3 but an SDK XML file of version 4 was encountered`

**说明**：命令行工具与 SDK 清单版本不完全同步的**警告**，一般可忽略。若要消掉：Android Studio → SDK Manager → 更新 **Android SDK Command-line Tools**。

### 推荐的干净重试流程

```powershell
# 1) 结束卡住的 Gradle/Java（按需）
# 2) 新开终端，确认镜像
echo $env:FLUTTER_STORAGE_BASE_URL   # 应为 https://storage.flutter-io.cn

cd D:\workplace\ycwl-erp-master\mobile
flutter clean
flutter pub get
flutter run -d <设备ID> -v --dart-define=API_BASE=http://192.168.1.29:18080/api/v1
```

后端需已启动：`go run ./cmd/erp-api`（默认 `http://<电脑局域网IP>:18080`）。
