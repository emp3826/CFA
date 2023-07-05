package com.github.kr328.clash.design.util

import android.app.Activity
import android.content.Context
import android.view.LayoutInflater
import android.view.ViewGroup
import androidx.annotation.DimenRes

val Context.layoutInflater: LayoutInflater
    get() = LayoutInflater.from(this)

val Context.root: ViewGroup?
    get() {
        return when (this) {
            is Activity -> {
                findViewById(android.R.id.content)
            }
            else -> {
                null
            }
        }
    }

fun Context.getPixels(@DimenRes resId: Int): Int {
    return resources.getDimensionPixelSize(resId)
}
