package com.github.kr328.clash.design

import android.content.Context
import android.view.View
import com.github.kr328.clash.core.model.Provider
import com.github.kr328.clash.design.adapter.ProviderAdapter
import com.github.kr328.clash.design.databinding.DesignProvidersBinding
import com.github.kr328.clash.design.util.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

class ProvidersDesign(
    context: Context,
    providers: List<Provider>,
) : Design<ProvidersDesign.Request>(context) {
    sealed class Request {
        data class Update(val index: Int, val provider: Provider) : Request()
    }

    private val binding = DesignProvidersBinding
        .inflate(context.layoutInflater, context.root, false)

    override val root: View
        get() = binding.root

    private val adapter = ProviderAdapter(context, providers) { index, provider ->
        requests.trySend(Request.Update(index, provider))
    }

    init {
        binding.self = this

        binding.activityBarLayout.applyFrom(context)

        binding.mainList.recyclerList.bindAppBarElevation(binding.activityBarLayout)
        binding.mainList.recyclerList.applyLinearAdapter(context, adapter)
    }
}
