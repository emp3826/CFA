import com.android.build.gradle.BaseExtension

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
            minSdk = 21
            targetSdk = 31

            versionName = "Custom"
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

        compileSdkVersion(defaultConfig.targetSdk!!)

        buildFeatures.apply {
            dataBinding {
                isEnabled = name != "hideapi"
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
