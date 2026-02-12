package com.trah.abnotify.ui

import android.app.AlertDialog
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.FrameLayout
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.trah.abnotify.AbnotifyApp
import com.trah.abnotify.databinding.FragmentMessagesBinding
import com.trah.abnotify.data.Message
import com.trah.abnotify.R
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Locale

class MessagesFragment : Fragment() {

    private var _binding: FragmentMessagesBinding? = null
    private val binding get() = _binding!!
    private lateinit var adapter: MessageAdapter
    
    // 待显示的消息ID（从通知点击进入时使用）
    private var pendingMessageId: String? = null

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentMessagesBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        adapter = MessageAdapter { message -> showMessageDetailDialog(message) }
        setupUI()
        observeMessages()
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }

    private var allMessages: List<Message> = emptyList()
    
    /**
     * 通过消息ID显示消息详情（从通知点击调用�?
     */
    fun showMessageById(messageId: String) {
        // 如果消息列表已加载，直接查找并显�?
        val message = allMessages.find { it.messageId == messageId }
        if (message != null) {
            showMessageDetailDialog(message)
        } else {
            // 消息列表还未加载，保存ID等待加载后显�?
            pendingMessageId = messageId
        }
    }

    private fun setupUI() {
        binding.recyclerView.layoutManager = LinearLayoutManager(context)
        binding.recyclerView.adapter = adapter

        binding.fabClear.setOnClickListener {
            showClearConfirmDialog()
        }

        // Search functionality
        binding.etSearch.addTextChangedListener(object : android.text.TextWatcher {
            override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) {}
            override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {}
            override fun afterTextChanged(s: android.text.Editable?) {
                filterMessages(s?.toString() ?: "")
            }
        })
    }

    private fun filterMessages(query: String) {
        val filtered = if (query.isBlank()) {
            allMessages
        } else {
            allMessages.filter { msg ->
                msg.title?.contains(query, ignoreCase = true) == true ||
                msg.body?.contains(query, ignoreCase = true) == true
            }
        }
        adapter.submitList(filtered)
        binding.tvEmpty.visibility = if (filtered.isEmpty()) View.VISIBLE else View.GONE
        binding.recyclerView.visibility = if (filtered.isEmpty()) View.GONE else View.VISIBLE
    }

    private fun observeMessages() {
        // Show loading
        binding.progressLoading.visibility = View.VISIBLE
        binding.recyclerView.visibility = View.GONE
        
        viewLifecycleOwner.lifecycleScope.launch {
            AbnotifyApp.getInstance().database.messageDao()
                .getAllMessages()
                .collectLatest { messages ->
                    // Hide loading
                    binding.progressLoading.visibility = View.GONE
                    binding.recyclerView.visibility = View.VISIBLE
                    
                    allMessages = messages
                    filterMessages(binding.etSearch.text?.toString() ?: "")
                    
                    // 检查是否有待显示的消息
                    pendingMessageId?.let { msgId ->
                        val message = messages.find { it.messageId == msgId }
                        if (message != null) {
                            pendingMessageId = null
                            showMessageDetailDialog(message)
                        }
                    }
                }
        }
    }

    /**
     * 显示消息详情对话�?
     */
    private fun showMessageDetailDialog(message: Message) {
        val dialogView = LayoutInflater.from(requireContext()).inflate(R.layout.dialog_message_detail, null)
        val tvTitle = dialogView.findViewById<TextView>(R.id.dialogTitle)
        val tvTime = dialogView.findViewById<TextView>(R.id.dialogTime)
        val tvBody = dialogView.findViewById<TextView>(R.id.dialogBody)
        val scrollBody = dialogView.findViewById<androidx.core.widget.NestedScrollView>(R.id.scrollBody)
        val btnCopy = dialogView.findViewById<TextView>(R.id.btnCopy)
        val btnClose = dialogView.findViewById<TextView>(R.id.btnClose)

        val dateFormat = SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault())
        
        tvTitle.text = message.title ?: "abnotify"
        tvTime.text = dateFormat.format(message.timestamp)
        
        // 格式化消息内�?
        val bodyText = formatMessageBody(message.body ?: "")
        tvBody.text = bodyText
        
        // 设置滚动区域最大高度为屏幕高度�?50%
        val displayMetrics = resources.displayMetrics
        val maxHeight = (displayMetrics.heightPixels * 0.5).toInt()
        scrollBody.post {
            if (scrollBody.height > maxHeight) {
                val params = scrollBody.layoutParams
                params.height = maxHeight
                scrollBody.layoutParams = params
            }
        }

        val alertDialog = AlertDialog.Builder(requireContext())
            .setView(dialogView)
            .create()

        alertDialog.window?.setBackgroundDrawableResource(android.R.color.transparent)

        btnCopy.setOnClickListener {
            copyToClipboard(message.body ?: "")
            Toast.makeText(requireContext(), "已复制到剪贴�?, Toast.LENGTH_SHORT).show()
        }

        btnClose.setOnClickListener {
            alertDialog.dismiss()
        }

        alertDialog.show()
    }

    /**
     * 格式化消息正�?- 尝试美化 JSON 或保持原�?
     */
    private fun formatMessageBody(body: String): String {
        // 尝试检测是否是 JSON 并格式化
        val trimmed = body.trim()
        if ((trimmed.startsWith("{") && trimmed.endsWith("}")) ||
            (trimmed.startsWith("[") && trimmed.endsWith("]"))) {
            return try {
                val gson = com.google.gson.GsonBuilder()
                    .setPrettyPrinting()
                    .create()
                val jsonElement = com.google.gson.JsonParser.parseString(trimmed)
                gson.toJson(jsonElement)
            } catch (e: Exception) {
                body
            }
        }
        return body
    }

    /**
     * 复制内容到剪贴板
     */
    private fun copyToClipboard(text: String) {
        val clipboard = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        val clip = ClipData.newPlainText("message", text)
        clipboard.setPrimaryClip(clip)
    }

    private fun showClearConfirmDialog() {
        showCleanDialog(
            title = "清空消息",
            message = "确定清空所有消息记录吗？此操作不可恢复�?,
            positiveText = "清空",
            onPositive = {
                viewLifecycleOwner.lifecycleScope.launch {
                    AbnotifyApp.getInstance().database.messageDao().deleteAll()
                }
            }
        )
    }

    private fun showCleanDialog(
        title: String,
        message: String,
        positiveText: String = "确定",
        negativeText: String? = "取消",
        customView: View? = null,
        onPositive: (() -> Unit)? = null
    ) {
        val dialogView = LayoutInflater.from(requireContext()).inflate(R.layout.dialog_clean, null)
        val tvTitle = dialogView.findViewById<TextView>(R.id.dialogTitle)
        val tvMessage = dialogView.findViewById<TextView>(R.id.dialogMessage)
        val btnPositive = dialogView.findViewById<TextView>(R.id.btnPositive)
        val btnNegative = dialogView.findViewById<TextView>(R.id.btnNegative)
        val customContainer = dialogView.findViewById<FrameLayout>(R.id.dialogCustomContainer)

        tvTitle.text = title
        tvMessage.text = message
        btnPositive.text = positiveText

        if (negativeText != null) {
            btnNegative.text = negativeText
            btnNegative.visibility = View.VISIBLE
        } else {
            btnNegative.visibility = View.GONE
        }

        if (customView != null) {
            customContainer.addView(customView)
            customContainer.visibility = View.VISIBLE
        }

        val alertDialog = AlertDialog.Builder(requireContext())
            .setView(dialogView)
            .create()

        alertDialog.window?.setBackgroundDrawableResource(android.R.color.transparent)

        btnPositive.setOnClickListener {
            onPositive?.invoke()
            alertDialog.dismiss()
        }

        btnNegative.setOnClickListener {
            alertDialog.dismiss()
        }

        alertDialog.show()
    }

    private class MessageAdapter(
        private val onItemClick: (Message) -> Unit
    ) : RecyclerView.Adapter<MessageAdapter.ViewHolder>() {
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
                tvTitle.text = message.title ?: "abnotify"
                tvBody.text = message.body ?: ""
                tvTime.text = dateFormat.format(message.timestamp)
                
                // 点击查看详情
                itemView.setOnClickListener {
                    onItemClick(message)
                }
            }
        }
    }
}
