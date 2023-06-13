package com.github.kr328.clash.service.clash.module

import android.app.Service
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkRequest
import android.net.NetworkCapabilities
import android.os.Build
import androidx.core.content.getSystemService
import com.github.kr328.clash.core.Clash
import com.github.kr328.clash.service.util.resolvePrimaryDns
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.channels.trySendBlocking
import kotlinx.coroutines.withContext

class NetworkObserveModule(service: Service) : Module<Network?>(service) {
    private data class Action(val type: Type, val network: Network) {
        enum class Type { Available, Lost, Changed }
    }

    private val connectivity = service.getSystemService<ConnectivityManager>()!!
    private val actions = Channel<Action>(Channel.UNLIMITED)
    private val request = NetworkRequest.Builder().apply {
        addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
        addCapability(NetworkCapabilities.NET_CAPABILITY_NOT_RESTRICTED)
    }.build()

    override suspend fun run() {
        try {
            val networks = mutableSetOf<Network>()

            while (true) {
                val action = actions.receive()

                val resolveDefault = when (action.type) {
                    Action.Type.Available -> {
                        networks.add(action.network)

                        true
                    }
                    Action.Type.Lost -> {
                        networks.remove(action.network)

                        true
                    }
                    Action.Type.Changed -> {
                        false
                    }
                }

                if (resolveDefault) {
                    val network = networks.maxByOrNull { net ->
                        connectivity.getNetworkCapabilities(net)?.let { cap ->
                            TRANSPORT_PRIORITY.indexOfFirst { cap.hasTransport(it) }
                        } ?: -1
                    }

                    enqueueEvent(network)
                }
            }
        } finally {
            withContext(NonCancellable) {
                enqueueEvent(null)
            }
        }
    }

    companion object {
        private val TRANSPORT_PRIORITY = sequence {
            yield(NetworkCapabilities.TRANSPORT_CELLULAR)
            yield(NetworkCapabilities.TRANSPORT_ETHERNET)
        }.toList()
    }
}
