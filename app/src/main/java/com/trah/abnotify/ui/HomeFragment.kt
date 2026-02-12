package com.trah.abnotify.ui

import android.content.BroadcastReceiver
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.graphics.Color
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.EditText
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import com.trah.abnotify.AbnotifyApp
import com.trah.abnotify.R
import com.trah.abnotify.databinding.FragmentHomeBinding
import com.trah.abnotify.service.WebSocketService
import com.trah.abnotify.util.KeepAliveHelper

class HomeFragment : Fragment() {

    private var _binding: FragmentHomeBinding? = null
    private val binding get() = _binding!!
    private val app by lazy { AbnotifyApp.getInstance() }

    private val connectionReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            if (intent?.action == WebSocketService.ACTION_CONNECTION_STATUS) {
                val connected = intent.getBooleanExtra(WebSocketService.EXTRA_CONNECTED, false)
                updateConnectionStatus(connected)
            }
        }
    }

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentHomeBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        setupUI()
        registerConnectionReceiver()
        animateCardsEntry()
    }

    private fun animateCardsEntry() {
        val views = listOf(
            binding.statusContainer,
            binding.cardTutorial
        )

        views.forEachIndexed { index, cardView ->
            cardView.apply {
                alpha = 0f
                translationY = 40f
                animate()
                    .alpha(1f)
                    .translationY(0f)
                    .setDuration(350)
                    .setStartDelay((index * 60).toLong())
                    .setInterpolator(android.view.animation.DecelerateInterpolator())
                    .start()
            }
        }
    }


    override fun onResume() {
        super.onResume()
        // Force refresh status if needed or check connection
        updateDeviceInfo()
    }

    override fun onDestroyView() {
        super.onDestroyView()
        requireContext().unregisterReceiver(connectionReceiver)
        _binding = null
    }

    private fun setupUI() {
        updateDeviceInfo()

        binding.btnRegister.setOnClickListener {
            (activity as? MainActivity)?.saveAndRegister()
        }

        binding.btnRefreshKey.setOnClickListener {
            showCleanDialog(
                title = "更换推送密�?,
                message = "更换密钥后原有的推送链接将立即失效。\n\n注意：更换后必须点击上方的“注册设�?/ 同步连接”按钮，否则无法接收新消息！",
                positiveText = "我知道了，更�?,
                onPositive = {
                    app.keyManager.regenerateDeviceKey()
                    updateDeviceInfo()
                    Toast.makeText(context, "密钥已更换，请点击注册按�?, Toast.LENGTH_LONG).show()
                }
            )
        }

        // Reset Encryption moved to SettingsFragment

        binding.btnCopyPushUrl.setOnClickListener {
            val url = "${app.keyManager.serverUrl.trimEnd('/')}/push/${app.keyManager.getDeviceKey()}"
            copyToClipboard("Push URL", url)
        }

        binding.cardAccessibility.setOnClickListener {
            startActivity(Intent(Settings.ACTION_ACCESSIBILITY_SETTINGS))
        }

        binding.cardAutoStart.setOnClickListener {
            // Open app settings to allow user to configure background running
            try {
                val intent = Intent(android.provider.Settings.ACTION_APPLICATION_DETAILS_SETTINGS)
                intent.data = android.net.Uri.parse("package:" + requireContext().packageName)
                startActivity(intent)
            } catch (e: Exception) {
                // Fallback to battery optimization settings
                KeepAliveHelper.requestIgnoreBatteryOptimization(requireContext())
            }
        }

        binding.cardTutorial.setOnClickListener {
            startActivity(Intent(requireContext(), TutorialActivity::class.java))
        }

        binding.tvPrivacyPolicy.setOnClickListener {
            showPrivacyDialog()
        }
        
        // Server URL configuration moved to SettingsFragment


    }

    private fun showPrivacyDialog() {
        val message = """
            1. 数据收集：Accnotify 不收集任何个人身份信息（�?IMEI、手机号等）�?
            2. 消息安全：所有推送消息均采用 RSA 端到端加密。服务器仅作为加密数据的搬运工，无法解密您的内容�?
            3. Push Key：它是您的唯一投递凭证，请妥善保管。一旦重置，旧链接将立即失效�?
            4. 无障碍服务：本应用申请无障碍权限仅用于增强后台运行稳定性及自动处理通知，不读取您的私人数据�?
            5. 免责声明：本软件为开源工具，请在中国法律允许范围内使用�?
        """.trimIndent()

        showCleanDialog(
            title = "隐私政策与服务协�?,
            message = message,
            positiveText = "我已知晓"
        )
    }

    private fun updateDeviceInfo() {
        val keyManager = app.keyManager
        binding.tvDeviceKey.text = keyManager.getDeviceKey() ?: "Unknown"
        binding.tvServerUrl.text = keyManager.serverUrl
    }

    private fun updateConnectionStatus(connected: Boolean) {
        val connectedColor = ContextCompat.getColor(requireContext(), R.color.status_clean_connected_bg)
        val connectedText = ContextCompat.getColor(requireContext(), R.color.status_clean_connected_text)
        val disconnectedColor = ContextCompat.getColor(requireContext(), R.color.status_clean_disconnected_bg)
        val disconnectedText = ContextCompat.getColor(requireContext(), R.color.status_clean_disconnected_text)

        if (connected) {
            binding.tvStatus.text = "服务已连�?
            binding.statusContainer.backgroundTintList = android.content.res.ColorStateList.valueOf(connectedColor)
            binding.tvStatus.setTextColor(connectedText)
            // Switch to connected Lottie animation
            binding.lottieStatus.setAnimation("anim_connected.json")
            binding.lottieStatus.playAnimation()
        } else {
            binding.tvStatus.text = "服务未连�?
            binding.statusContainer.backgroundTintList = android.content.res.ColorStateList.valueOf(disconnectedColor)
            binding.tvStatus.setTextColor(disconnectedText)
            // Switch to disconnected Lottie animation
            binding.lottieStatus.setAnimation("anim_disconnected.json")
            binding.lottieStatus.playAnimation()
        }
    }


    private fun copyToClipboard(label: String, text: String) {
        val clipboard = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        val clip = ClipData.newPlainText(label, text)
        clipboard.setPrimaryClip(clip)
        Toast.makeText(context, "已复�?, Toast.LENGTH_SHORT).show()
    }

    private fun registerConnectionReceiver() {
        val filter = IntentFilter(WebSocketService.ACTION_CONNECTION_STATUS)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            requireContext().registerReceiver(connectionReceiver, filter, Context.RECEIVER_EXPORTED)
        } else {
            requireContext().registerReceiver(connectionReceiver, filter)
        }
    }

    // Server Dialog logic moved to SettingsFragment

    private fun showCleanDialog(
        title: String,
        message: String,
        positiveText: String = "OK",
        negativeText: String? = "取消",
        customView: View? = null,
        onPositive: (() -> Unit)? = null
    ) {
        val dialogView = LayoutInflater.from(requireContext()).inflate(R.layout.dialog_clean, null)
        val tvTitle = dialogView.findViewById<android.widget.TextView>(R.id.dialogTitle)
        val tvMessage = dialogView.findViewById<android.widget.TextView>(R.id.dialogMessage)
        val btnPositive = dialogView.findViewById<android.widget.TextView>(R.id.btnPositive)
        val btnNegative = dialogView.findViewById<android.widget.TextView>(R.id.btnNegative)
        val customContainer = dialogView.findViewById<android.widget.FrameLayout>(R.id.dialogCustomContainer)

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
            // If custom view is present, message acts as subtitle, often good to keep visible
        }

        val alertDialog = AlertDialog.Builder(requireContext())
            .setView(dialogView)
            .create()

        // Make window transparent so rounded corners show
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
}

