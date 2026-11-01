package com.mushroomserchz.trae.item;

import net.minecraft.entity.LivingEntity;
import net.minecraft.entity.player.PlayerEntity;
import net.minecraft.item.ItemStack;
import net.minecraft.item.SwordItem;
import net.minecraft.item.ToolMaterial;
import net.minecraft.particle.ParticleTypes;
import net.minecraft.server.world.ServerWorld;
import net.minecraft.util.math.Box;

import java.util.List;

public class GodslayerSwordItem extends SwordItem {
    public GodslayerSwordItem(ToolMaterial toolMaterial, int attackDamage, float attackSpeed, Settings settings) {
        super(toolMaterial, attackDamage, attackSpeed, settings);
    }

    @Override
    public boolean postHit(ItemStack stack, LivingEntity target, LivingEntity attacker) {
        if (attacker.getWorld().isClient()) {
            return super.postHit(stack, target, attacker);
        }

        float lethalDamage = 9999.0F;
        target.damage(target.getDamageSources().magic(), lethalDamage);
        target.getWorld().spawnParticles(
                ParticleTypes.SOUL_FIRE_FLAME,
                target.getX(),
                target.getBodyY(0.5D),
                target.getZ(),
                80,
                0.75,
                0.75,
                0.75,
                0.2
        );

        Box blastArea = target.getBoundingBox().expand(8.0D);
        List<LivingEntity> nearby = target.getWorld().getEntitiesByClass(
                LivingEntity.class,
                blastArea,
                entity -> entity != attacker && entity != target
        );

        for (LivingEntity nearbyEntity : nearby) {
            nearbyEntity.damage(nearbyEntity.getDamageSources().magic(), lethalDamage);
            nearbyEntity.getWorld().spawnParticles(
                    ParticleTypes.ENCHANT,
                    nearbyEntity.getX(),
                    nearbyEntity.getBodyY(0.5D),
                    nearbyEntity.getZ(),
                    50,
                    0.8,
                    0.8,
                    0.8,
                    0.05
            );
        }

        if (attacker instanceof PlayerEntity player) {
            player.getWorld().spawnParticles(
                    ParticleTypes.END_ROD,
                    target.getX(),
                    target.getBodyY(0.5D),
                    target.getZ(),
                    100,
                    1.0,
                    1.0,
                    1.0,
                    0.12
            );
        }

        return super.postHit(stack, target, attacker);
    }
}
