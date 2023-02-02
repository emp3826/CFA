package com.github.kr328.clash.design.adapter

import android.content.Context
import android.view.View
import android.view.ViewGroup
import androidx.recyclerview.widget.RecyclerView
import com.github.kr328.clash.core.model.Provider
import com.github.kr328.clash.design.databinding.AdapterProviderBinding
import com.github.kr328.clash.design.model.ProviderState
import com.github.kr328.clash.design.util.layoutInflater

class ProviderAdapter(
    private val context: Context,
    providers: List<Provider>,
    private val requestUpdate: (Int, Provider) -> Unit,
) : RecyclerView.Adapter<ProviderAdapter.Holder>() {
    class Holder(val binding: AdapterProviderBinding) : RecyclerView.ViewHolder(binding.root)

    val states = providers.map { ProviderState(it, it.updatedAt, false) }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): Holder {
        return Holder(
            AdapterProviderBinding
                .inflate(context.layoutInflater, parent, false)
        )
    }

    override fun onBindViewHolder(holder: Holder, position: Int) {
        val state = states[position]

        holder.binding.provider = state.provider
        holder.binding.state = state
    }

    override fun getItemCount(): Int {
        return states.size
    }
}
