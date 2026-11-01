package com.mushroomserchz.trae.item;

import com.mushroomserchz.trae.TraeMod;
import net.minecraft.item.ArmorItem;
import net.minecraft.item.Item;
import net.minecraft.item.Rarity;
import net.minecraft.item.ToolMaterials;
import net.minecraft.registry.Registries;
import net.minecraft.registry.Registry;

public final class ModItems {
    public static final Item GODS_EDGE = registerItem(
            "gods_edge",
            new GodslayerSwordItem(ToolMaterials.NETHERITE, 2048, -2.2F,
                    new Item.Settings().fireproof().rarity(Rarity.EPIC))
    );

    public static final Item ABYSSAL_HELMET = registerItem(
            "abyssal_helmet",
            new ArmorItem(ModArmorMaterials.OVERLORD, ArmorItem.Type.HELMET,
                    new Item.Settings().fireproof().rarity(Rarity.EPIC))
    );

    public static final Item ABYSSAL_CHESTPLATE = registerItem(
            "abyssal_chestplate",
            new ArmorItem(ModArmorMaterials.OVERLORD, ArmorItem.Type.CHESTPLATE,
                    new Item.Settings().fireproof().rarity(Rarity.EPIC))
    );

    public static final Item ABYSSAL_LEGGINGS = registerItem(
            "abyssal_leggings",
            new ArmorItem(ModArmorMaterials.OVERLORD, ArmorItem.Type.LEGGINGS,
                    new Item.Settings().fireproof().rarity(Rarity.EPIC))
    );

    public static final Item ABYSSAL_BOOTS = registerItem(
            "abyssal_boots",
            new ArmorItem(ModArmorMaterials.OVERLORD, ArmorItem.Type.BOOTS,
                    new Item.Settings().fireproof().rarity(Rarity.EPIC))
    );

    private ModItems() {
    }

    public static void register() {
    }

    private static Item registerItem(String name, Item item) {
        return Registry.register(Registries.ITEM, TraeMod.id(name), item);
    }
}
