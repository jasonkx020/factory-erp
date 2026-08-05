# 加工厂 ERP · 员工 App（Flutter）

Android / iOS **唯一**员工现场端。与管理端共用 `/api/v1` 与 IAM；登录 `client_type=mobile`。

## 环境要求

- Flutter SDK（本仓库常用路径示例：`D:\flutter`）
- Android SDK / JDK 17（Android Studio 自带 JBR 即可）
- 真机或模拟器；API 默认端口 `:18080`

## 开发

```powershell
cd mobile
flutter pub get

# 国内网络强烈建议先设镜像（新开终端也会继承 User 级环境变量）
$env:FLUTTER_STORAGE_BASE_URL = "https://storage.flutter-io.cn"
$env:PUB_HOSTED_URL = "https://pub.flutter-io.cn"

flutter run -d <设备ID> --dart-define=API_BASE=http://192.168.x.x:18080/api/v1
```

演示账号：`admin` / `admin123`

持久化镜像（PowerShell，只需执行一次）：

```powershell
[Environment]::SetEnvironmentVariable("FLUTTER_STORAGE_BASE_URL", "https://storage.flutter-io.cn", "User")
[Environment]::SetEnvironmentVariable("PUB_HOSTED_URL", "https://pub.flutter-io.cn", "User")
```

查看完整编译过程：

```powershell
flutter run -d <设备ID> -v --dart-define=API_BASE=http://192.168.x.x:18080/api/v1
# 或单独看 Gradle：
cd android
.\gradlew.bat assembleDebug --info --stacktrace
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
