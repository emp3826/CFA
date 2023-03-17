package com.github.kr328.clash.remote

import android.content.Context
import android.content.Intent
import com.github.kr328.clash.common.Global
import com.github.kr328.clash.common.util.intent
import com.github.kr328.clash.util.ApplicationObserver
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.launch

object Remote {
    val broadcasts: Broadcasts = Broadcasts(Global.application)
    val service: Service = Service(Global.application) {
        ApplicationObserver.createdActivities.forEach { it.finish() }
    }

    private val visible = Channel<Boolean>(Channel.CONFLATED)

    fun launch() {
        ApplicationObserver.attach(Global.application)

        ApplicationObserver.onVisibleChanged { visible.trySend(it) }

        Global.launch(Dispatchers.IO) {
            run()
        }
    }

    private suspend fun run() {
        while (true) {
            if (visible.receive()) {
                service.bind()
                broadcasts.register()
            } else {
                service.unbind()
                broadcasts.unregister()
            }
        }
    }
}
