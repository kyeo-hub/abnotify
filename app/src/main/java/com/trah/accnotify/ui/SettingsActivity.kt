package com.trah.accnotify.ui

import android.os.Bundle
import android.view.MenuItem
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.preference.Preference
import androidx.preference.PreferenceFragmentCompat
import com.trah.accnotify.BuildConfig
import com.trah.accnotify.AccnotifyApp
import com.trah.accnotify.R

class SettingsActivity : AppCompatActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_settings)

        supportActionBar?.apply {
            title = getString(R.string.settings)
            setDisplayHomeAsUpEnabled(true)
        }

        supportFragmentManager
            .beginTransaction()
            .replace(R.id.settings_container, SettingsFragment())
            .commit()
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

    class SettingsFragment : PreferenceFragmentCompat() {

        override fun onCreatePreferences(savedInstanceState: Bundle?, rootKey: String?) {
            setPreferencesFromResource(R.xml.preferences, rootKey)

            // Version
            findPreference<Preference>("version")?.summary = BuildConfig.VERSION_NAME

            // Keep Alive Settings
            setupKeepAlivePreferences()

            // Regenerate device key only
            findPreference<Preference>("regenerate_key")?.setOnPreferenceClickListener {
                showRegenerateKeyDialog()
                true
            }

            // Full reset (device key + RSA keys)
            findPreference<Preference>("regenerate_all")?.setOnPreferenceClickListener {
                showFullResetDialog()
                true
            }
        }

        private fun setupKeepAlivePreferences() {
            val context = requireContext()
            
            findPreference<Preference>("keep_alive_status")?.setOnPreferenceClickListener {
                showKeepAliveStatusDialog()
                true
            }

            findPreference<Preference>("battery_optimization")?.setOnPreferenceClickListener {
                com.trah.accnotify.util.KeepAliveHelper.requestIgnoreBatteryOptimization(context)
                true
            }

            findPreference<Preference>("pref_keep_alive_auto_start")?.setOnPreferenceClickListener {
                com.trah.accnotify.util.KeepAliveHelper.openAutoStartSettings(context)
                true
            }
            
            findPreference<Preference>("pref_keep_alive_accessibility")?.setOnPreferenceClickListener {
                com.trah.accnotify.util.KeepAliveHelper.openAccessibilitySettings(context)
                true
            }

            findPreference<Preference>("background_settings")?.setOnPreferenceClickListener {
                com.trah.accnotify.util.KeepAliveHelper.openBackgroundSettings(context)
                true
            }

            findPreference<Preference>("notification_permission")?.setOnPreferenceClickListener {
                com.trah.accnotify.util.KeepAliveHelper.openNotificationSettings(context)
                true
            }

            findPreference<Preference>("exact_alarm")?.setOnPreferenceClickListener {
                com.trah.accnotify.util.KeepAliveHelper.requestExactAlarmPermission(context)
                true
            }

            findPreference<Preference>("app_settings")?.setOnPreferenceClickListener {
                com.trah.accnotify.util.KeepAliveHelper.openAppSettings(context)
                true
            }
        }

        override fun onResume() {
            super.onResume()
            updateSummaries()
        }

        private fun updateSummaries() {
            val context = requireContext()
            val helper = com.trah.accnotify.util.KeepAliveHelper
            
            findPreference<Preference>("battery_optimization")?.summary = 
                if (helper.isIgnoringBatteryOptimizations(context)) "✓ 已允许" else "✗ 未允许 (点击跳转)"
            
            findPreference<Preference>("pref_keep_alive_accessibility")?.summary = 
                if (helper.isAccessibilityServiceEnabled(context)) "✓ 已开启" else "✗ 未开启 (点击跳转)"
                
            if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.S) {
                findPreference<Preference>("exact_alarm")?.summary = 
                    if (helper.canScheduleExactAlarms(context)) "✓ 已允许" else "✗ 未允许 (点击跳转)"
            }
        }

        private fun showKeepAliveStatusDialog() {
            val context = requireContext()
            val helper = com.trah.accnotify.util.KeepAliveHelper
            
            val summary = helper.getStatusSummary(context)
            val actions = helper.getRecommendedActions()
            
            val message = StringBuilder(summary)
            message.append("\n${getString(R.string.about)}建议操作:\n")
            actions.forEach { message.append("• $it\n") }
            
            AlertDialog.Builder(context)
                .setTitle("保活状态及建议")
                .setMessage(message.toString())
                .setPositiveButton(R.string.confirm, null)
                .show()
        }


        private fun showRegenerateKeyDialog() {
            AlertDialog.Builder(requireContext())
                .setTitle("重新生成设备码")
                .setMessage("将生成新的设备码，需要重新注册到服务器。\n\n加密密钥保持不变，已加密的消息仍可解密。")
                .setPositiveButton(R.string.confirm) { _, _ ->
                    val keyManager = AccnotifyApp.getInstance().keyManager
                    keyManager.regenerateDeviceKey()
                    Toast.makeText(requireContext(), "设备码已更新，请重新注册", Toast.LENGTH_SHORT).show()
                    activity?.recreate()
                }
                .setNegativeButton(R.string.cancel, null)
                .show()
        }

        private fun showFullResetDialog() {
            AlertDialog.Builder(requireContext())
                .setTitle("完全重置")
                .setMessage("将更换设备码和加密密钥！\n\n• 需要重新注册到服务器\n• 已加密的消息将无法解密\n\n确定要完全重置吗？")
                .setPositiveButton("确定重置") { _, _ ->
                    val keyManager = AccnotifyApp.getInstance().keyManager
                    keyManager.regenerateAllKeys()
                    Toast.makeText(requireContext(), "已完全重置，请重新注册", Toast.LENGTH_SHORT).show()
                    activity?.recreate()
                }
                .setNegativeButton(R.string.cancel, null)
                .show()
        }
    }
}
