plugins {
    kotlin("android")
    id("com.android.library")
}

dependencies {
    implementation(libs.kotlin.coroutine)
    implementation(libs.androidx.core)
}
