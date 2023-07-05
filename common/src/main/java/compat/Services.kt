package com.github.kr328.clash.common.compat

import android.content.Context
import android.content.Intent

fun Context.startForegroundServiceCompat(intent: Intent) {
    startForegroundService(intent)
}
