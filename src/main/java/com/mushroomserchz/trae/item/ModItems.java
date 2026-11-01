package com.mushroomserchz.trae;

import com.mushroomserchz.trae.item.ModItems;
import net.fabricmc.fabric.api.entity.event.v1.ServerLivingEntityEvents;
import net.fabricmc.fabric.api.event.lifecycle.v1.ServerTickEvents;
import net.minecraft.entity.EquipmentSlot;
import net.minecraft.entity.effect.StatusEffectInstance;
import net.minecraft.entity.effect.StatusEffects;
import net.minecraft.entity.player.PlayerEntity;
import net.minecraft.item.Item;

public final class TraeDamageController {
    private TraeDamageController() {
    }

    public static void register() {
        ServerLivingEntityEvents.ALLOW_DAMAGE.register((entity, source, amount) -> {
            if (entity instanceof PlayerEntity player && isFullSet(player)) {
                return false;
            }
            return true;
        });

        ServerTickEvents.END_SERVER_TICK.register(server -> {
            for (PlayerEntity player : server.getPlayerManager().getPlayerList()) {
                if (!isFullSet(player)) {
                    continue;
                }

                player.setHealth(player.getMaxHealth());
                player.getHungerManager().setFoodLevel(20);
                player.getHungerManager().setSaturationLevelClient(20.0F);
                player.addStatusEffect(new StatusEffectInstance(StatusEffects.REGENERATION, 30, 4, false, false));
                player.addStatusEffect(new StatusEffectInstance(StatusEffects.RESISTANCE, 30, 3, false, false));
                player.addStatusEffect(new StatusEffectInstance(StatusEffects.STRENGTH, 30, 4, false, false));
                player.addStatusEffect(new StatusEffectInstance(StatusEffects.SPEED, 30, 2, false, false));
            }
        });
    }

    private static boolean isFullSet(PlayerEntity player) {
        return isArmorPiece(player, EquipmentSlot.HEAD, ModItems.ABYSSAL_HELMET)
                && isArmorPiece(player, EquipmentSlot.CHEST, ModItems.ABYSSAL_CHESTPLATE)
                && isArmorPiece(player, EquipmentSlot.LEGS, ModItems.ABYSSAL_LEGGINGS)
                && isArmorPiece(player, EquipmentSlot.FEET, ModItems.ABYSSAL_BOOTS);
    }

    private static boolean isArmorPiece(PlayerEntity player, EquipmentSlot slot, Item item) {
        return player.getEquippedStack(slot).isOf(item);
    }
}
