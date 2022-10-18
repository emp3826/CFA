import com.android.build.gradle.AppExtension
import com.android.build.gradle.BaseExtension
import java.util.*

buildscript {
    repositories {
        mavenCentral()
        google()
        maven("https://maven.kr328.app/releases")
    }
    dependencies {
        classpath(libs.build.android)
        classpath(libs.build.kotlin.common)
        classpath(libs.build.kotlin.serialization)
        classpath(libs.build.ksp)
        classpath(libs.build.golang)
    }
}

subprojects {
    repositories {
        mavenCentral()
        google()
        maven("https://maven.kr328.app/releases")
    }

    val isApp = name == "app"

    apply(plugin = if (isApp) "com.android.application" else "com.android.library")

    extensions.configure<BaseExtension> {
        defaultConfig {
            if (isApp) {
                applicationId = "com.github.kr328.clash"
            }

            minSdk = 21
            targetSdk = 31

            versionName = "0.0.0"
            versionCode = 16418

            resValue("string", "release_name", "v$versionName")
            resValue("integer", "release_code", "$versionCode")

            externalNativeBuild {
                cmake {
                    abiFilters("arm64-v8a")
                }
            }

            if (!isApp) {
                consumerProguardFiles("consumer-rules.pro")
            } else {
                setProperty("archivesBaseName", "clash")
            }
        }

        ndkVersion = "25.0.8775105"

        compileSdkVersion(defaultConfig.targetSdk!!)

        productFlavors {
            flavorDimensions("feature")

            create("foss") {
                isDefault = true
                dimension = flavorDimensionList[0]
            }
        }

        signingConfigs {
            val keystore = rootProject.file("signing.properties")
            if (keystore.exists()) {
                create("release") {
                    val prop = Properties().apply {
                        keystore.inputStream().use(this::load)
                    }

                    storeFile = rootProject.file(prop.getProperty("keystore.path")!!)
                    storePassword = prop.getProperty("keystore.password")!!
                    keyAlias = prop.getProperty("key.alias")!!
                    keyPassword = prop.getProperty("key.password")!!
                }
            }
        }

        buildTypes {
            named("release") {
                isMinifyEnabled = isApp
                isShrinkResources = isApp
                signingConfig = signingConfigs.findByName("release")
                proguardFiles(
                    getDefaultProguardFile("proguard-android-optimize.txt"),
                    "proguard-rules.pro"
                )
            }
        }

        buildFeatures.apply {
            dataBinding {
                isEnabled = name != "hideapi"
            }
        }

        if (isApp) {
            this as AppExtension

            splits {
                abi {
                    isEnable = true
                }
            }
        }
    }
}

task("clean", type = Delete::class) {
    delete(rootProject.buildDir)
}

tasks.wrapper {
    distributionType = Wrapper.DistributionType.ALL
}
