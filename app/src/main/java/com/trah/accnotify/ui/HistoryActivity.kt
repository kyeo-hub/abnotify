package com.trah.accnotify.ui

import android.os.Bundle
import android.view.MenuItem
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import com.google.android.material.floatingactionbutton.FloatingActionButton
import com.trah.accnotify.AccnotifyApp
import com.trah.accnotify.R
import com.trah.accnotify.data.Message
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Locale

class HistoryActivity : AppCompatActivity() {

    private lateinit var recyclerView: RecyclerView
    private lateinit var emptyView: TextView
    private lateinit var fabClear: FloatingActionButton
    private val adapter = MessageAdapter()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_history)

        supportActionBar?.apply {
            title = getString(R.string.history)
            setDisplayHomeAsUpEnabled(true)
        }

        recyclerView = findViewById(R.id.recyclerView)
        emptyView = findViewById(R.id.tvEmpty)
        fabClear = findViewById(R.id.fabClear)

        recyclerView.layoutManager = LinearLayoutManager(this)
        recyclerView.adapter = adapter

        fabClear.setOnClickListener {
            showClearConfirmDialog()
        }

        observeMessages()
    }

    override fun onOptionsItemSelected(item: MenuItem): Boolean {
        return when (item.itemId) {
            android.R.id.home -> {
                finish()
                true
            }
            else -> super.onOptionsItemSelected(item)
        }
    }

    private fun observeMessages() {
        lifecycleScope.launch {
            AccnotifyApp.getInstance().database.messageDao()
                .getAllMessages()
                .collectLatest { messages ->
                    adapter.submitList(messages)
                    emptyView.visibility = if (messages.isEmpty()) View.VISIBLE else View.GONE
                    recyclerView.visibility = if (messages.isEmpty()) View.GONE else View.VISIBLE
                }
        }
    }

    private fun showClearConfirmDialog() {
        AlertDialog.Builder(this)
            .setMessage(R.string.clear_history_confirm)
            .setPositiveButton(R.string.confirm) { _, _ ->
                lifecycleScope.launch {
                    AccnotifyApp.getInstance().database.messageDao().deleteAll()
                }
            }
            .setNegativeButton(R.string.cancel, null)
            .show()
    }

    // Message Adapter
    private class MessageAdapter : RecyclerView.Adapter<MessageAdapter.ViewHolder>() {
        private var messages: List<Message> = emptyList()
        private val dateFormat = SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault())

        fun submitList(list: List<Message>) {
            messages = list
            notifyDataSetChanged()
        }

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
            val view = LayoutInflater.from(parent.context)
                .inflate(R.layout.item_message, parent, false)
            return ViewHolder(view)
        }

        override fun onBindViewHolder(holder: ViewHolder, position: Int) {
            holder.bind(messages[position])
        }

        override fun getItemCount(): Int = messages.size

        inner class ViewHolder(itemView: View) : RecyclerView.ViewHolder(itemView) {
            private val tvTitle: TextView = itemView.findViewById(R.id.tvTitle)
            private val tvBody: TextView = itemView.findViewById(R.id.tvBody)
            private val tvTime: TextView = itemView.findViewById(R.id.tvTime)

            fun bind(message: Message) {
                tvTitle.text = message.title ?: "Accnotify"
                tvBody.text = message.body ?: ""
                tvTime.text = dateFormat.format(message.timestamp)
            }
        }
    }
}
