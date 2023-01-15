package com.github.kr328.clash.design.adapter

import android.content.Context
import android.view.ViewGroup
import androidx.recyclerview.widget.RecyclerView
import com.github.kr328.clash.design.databinding.AdapterProfileProviderBinding
import com.github.kr328.clash.design.model.ProfileProvider
import com.github.kr328.clash.design.util.layoutInflater

class ProfileProviderAdapter(
    private val context: Context,
    private val select: (ProfileProvider) -> Unit,
) : RecyclerView.Adapter<ProfileProviderAdapter.Holder>() {
    class Holder(val binding: AdapterProfileProviderBinding) : RecyclerView.ViewHolder(binding.root)

    var providers: List<ProfileProvider> = emptyList()

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): Holder {
        return Holder(
            AdapterProfileProviderBinding.inflate(
                context.layoutInflater,
                parent,
                false
            )
        )
    }

    override fun onBindViewHolder(holder: Holder, position: Int) {
        val current = providers[position]
        val binding = holder.binding

        binding.provider = current

        binding.root.apply {
            setOnClickListener {
                select(current)
            }
        }
    }

    override fun getItemCount(): Int {
        return providers.size
    }
}
